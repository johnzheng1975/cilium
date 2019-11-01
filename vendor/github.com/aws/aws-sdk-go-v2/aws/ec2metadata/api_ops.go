package ec2metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
)

// GetMetadata uses the path provided to request information from the EC2
// instance metdata service. The content will be returned as a string, or
// error if the request failed.
func (c *Client) GetMetadata(p string) (string, error) {
	op := &aws.Operation{
		Name:       "GetMetadata",
		HTTPMethod: "GET",
		HTTPPath:   suffixPath("/meta-data", p),
	}

	output := &metadataOutput{}
	req := c.NewRequest(op, nil, output)

	return output.Content, req.Send()
}

// GetUserData returns the userdata that was configured for the service. If
// there is no user-data setup for the EC2 instance a "NotFoundError" error
// code will be returned.
func (c *Client) GetUserData() (string, error) {
	op := &aws.Operation{
		Name:       "GetUserData",
		HTTPMethod: "GET",
		HTTPPath:   "/user-data",
	}

	output := &metadataOutput{}
	req := c.NewRequest(op, nil, output)
	req.Handlers.UnmarshalError.PushBack(func(r *aws.Request) {
		if r.HTTPResponse.StatusCode == http.StatusNotFound {
			r.Error = awserr.New("NotFoundError", "user-data not found", r.Error)
		}
	})

	return output.Content, req.Send()
}

// GetDynamicData uses the path provided to request information from the EC2
// instance metadata service for dynamic data. The content will be returned
// as a string, or error if the request failed.
func (c *Client) GetDynamicData(p string) (string, error) {
	op := &aws.Operation{
		Name:       "GetDynamicData",
		HTTPMethod: "GET",
		HTTPPath:   suffixPath("/dynamic", p),
	}

	output := &metadataOutput{}
	req := c.NewRequest(op, nil, output)

	return output.Content, req.Send()
}

// GetInstanceIdentityDocument retrieves an identity document describing an
// instance. Error is returned if the request fails or is unable to parse
// the response.
func (c *Client) GetInstanceIdentityDocument() (EC2InstanceIdentityDocument, error) {
	resp, err := c.GetDynamicData("instance-identity/document")
	if err != nil {
		return EC2InstanceIdentityDocument{},
			awserr.New("EC2MetadataRequestError",
				"failed to get EC2 instance identity document", err)
	}

	doc := EC2InstanceIdentityDocument{}
	if err := json.NewDecoder(strings.NewReader(resp)).Decode(&doc); err != nil {
		return EC2InstanceIdentityDocument{},
			awserr.New("SerializationError",
				"failed to decode EC2 instance identity document", err)
	}

	return doc, nil
}

// IAMInfo retrieves IAM info from the metadata API
func (c *Client) IAMInfo() (EC2IAMInfo, error) {
	resp, err := c.GetMetadata("iam/info")
	if err != nil {
		return EC2IAMInfo{},
			awserr.New("EC2MetadataRequestError",
				"failed to get EC2 IAM info", err)
	}

	info := EC2IAMInfo{}
	if err := json.NewDecoder(strings.NewReader(resp)).Decode(&info); err != nil {
		return EC2IAMInfo{},
			awserr.New("SerializationError",
				"failed to decode EC2 IAM info", err)
	}

	if info.Code != "Success" {
		errMsg := fmt.Sprintf("failed to get EC2 IAM Info (%s)", info.Code)
		return EC2IAMInfo{},
			awserr.New("EC2MetadataError", errMsg, nil)
	}

	return info, nil
}

// Region returns the region the instance is running in.
func (c *Client) Region() (string, error) {
	resp, err := c.GetMetadata("placement/availability-zone")
	if err != nil {
		return "", err
	}

	// returns region without the suffix. Eg: us-west-2a becomes us-west-2
	return resp[:len(resp)-1], nil
}

// Available returns if the application has access to the EC2 Instance Metadata
// service.  Can be used to determine if application is running within an EC2
// Instance and the metadata service is available.
func (c *Client) Available() bool {
	if _, err := c.GetMetadata("instance-id"); err != nil {
		return false
	}

	return true
}

// An EC2IAMInfo provides the shape for unmarshaling
// an IAM info from the metadata API
type EC2IAMInfo struct {
	Code               string
	LastUpdated        time.Time
	InstanceProfileArn string
	InstanceProfileID  string
}

// An EC2InstanceIdentityDocument provides the shape for unmarshaling
// an instance identity document
type EC2InstanceIdentityDocument struct {
	DevpayProductCodes      []string  `json:"devpayProductCodes"`
	MarketplaceProductCodes []string  `json:"marketplaceProductCodes"`
	AvailabilityZone        string    `json:"availabilityZone"`
	PrivateIP               string    `json:"privateIp"`
	Version                 string    `json:"version"`
	Region                  string    `json:"region"`
	InstanceID              string    `json:"instanceId"`
	BillingProducts         []string  `json:"billingProducts"`
	InstanceType            string    `json:"instanceType"`
	AccountID               string    `json:"accountId"`
	PendingTime             time.Time `json:"pendingTime"`
	ImageID                 string    `json:"imageId"`
	KernelID                string    `json:"kernelId"`
	RamdiskID               string    `json:"ramdiskId"`
	Architecture            string    `json:"architecture"`
}

func suffixPath(base, add string) string {
	reqPath := path.Join(base, add)
	if len(add) != 0 && add[len(add)-1] == '/' {
		reqPath += "/"
	}
	return reqPath
}