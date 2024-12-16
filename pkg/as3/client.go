/*
Copyright 2021 The Kube-OVN AS3 Controller Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package as3

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// Client represents an AS3 client
type Client struct {
	url      string
	username string
	password string
	//use to request other not as3 api
	host string
	*http.Client
	sync.Mutex
}

func NewClient(ip, username, password string, insecure bool) *Client {
	client := &Client{
		url:      fmt.Sprintf("https://%s/mgmt/shared/appsvcs/declare/", ip),
		username: username,
		password: password,
		host:     "https://" + ip,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
	if insecure {
		client.Client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	return client
}

func (c *Client) post(data interface{}, tenants ...string) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	url := c.host + "/mgmt/shared/appsvcs/declare/" + strings.Join(tenants, ",")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		klog.Errorf("Failed to create AS3 request: %v", err)
		return err
	}
	klog.V(3).Infof("method = %s, url = %s, body = %s", req.Method, req.URL.String(), string(b))
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("Failed to call AS3 API: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return err
	}
	klog.V(3).Infof("response: body = %s", string(respBody))
	var response map[string]interface{}
	if err = json.Unmarshal(respBody, &response); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return err
	}
	return handleResponse(resp.StatusCode, response)
}

func (c *Client) Get(partition string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, c.url+partition, nil)
	if err != nil {
		klog.Errorf("Failed to create AS3 request: %v", err)
		return "", err
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("Failed to call AS3 API: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return "", err
	}
	//Common tenant isn't exist, body is "" == (as3 do not set)
	if err == nil && resp.StatusCode > 199 && resp.StatusCode < 299 && string(respBody) == "" {
		return "{}", nil
	}
	//specified Tenant(s) not found in declaration
	if resp.StatusCode == 404 {
		return "{}", nil
	}
	var response map[string]interface{}
	if err = json.Unmarshal(respBody, &response); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return "", err
	}
	if err = handleResponse(resp.StatusCode, response); err != nil {
		return "", err
	}

	return string(respBody), nil
}

func (c *Client) PostRaw(data []byte) error {
	req, err := http.NewRequest(http.MethodPost, c.url, bytes.NewBuffer(data))
	if err != nil {
		klog.Errorf("Failed to create AS3 request: %v", err)
		return err
	}
	klog.Infof("method = %s, url = %s, body = %s", req.Method, req.URL.String(), string(data))
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("Failed to call AS3 API: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return err
	}
	klog.Infof("response: body = %s", string(respBody))
	var response map[string]interface{}
	if err = json.Unmarshal(respBody, &response); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return err
	}

	return handleResponse(resp.StatusCode, response)
}

func (c *Client) patch(patchItems ...PatchItem) error {
	if len(patchItems) == 0 {
		klog.Info("no data need to patch")
		return nil
	}
	b, err := json.Marshal(patchItems)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, c.url, bytes.NewBuffer(b))
	if err != nil {
		klog.Errorf("Failed to create AS3 request: %v", err)
		return err
	}
	klog.Infof("request: method = %s, url = %s, body = %s", req.Method, req.URL.String(), string(b))
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("Failed to call AS3 API: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return err
	}
	klog.Infof("response: body = %s", string(respBody))
	var response map[string]interface{}
	if err = json.Unmarshal(respBody, &response); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return err
	}
	return handleResponse(resp.StatusCode, response)
}

func handleResponse(statusCode int, response map[string]interface{}) error {
	switch statusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		return nil
	}
	if results, ok := (response["results"]).([]interface{}); ok {
		for _, value := range results {
			v := value.(map[string]interface{})
			klog.Errorf("Response from BIG-IP: code = %v, tenant = %v, message = %v, response = %v", v["code"], v["tenant"], v["message"], v["response"])
		}
	} else if err, ok := (response["error"]).(map[string]interface{}); ok {
		//klog.Errorf("Big-IP Responded with error code: %v", err["code"])
		return fmt.Errorf("Big-IP Responded with error code: %v", err["code"])
	} else {
		//klog.Errorf("Big-IP Responded with code: %v", response["code"])
		return fmt.Errorf("Big-IP Responded with status code: %v", response["code"])
	}
	return fmt.Errorf("AS3 responds with status code: %d - %s", statusCode, http.StatusText(statusCode))
}

func (c *Client) patchF5Reource(obj interface{}, url string) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	url = c.host + url
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(data))
	if err != nil {
		klog.Errorf("Failed to create BIG-IP resouce request: %v", err)
		return err
	}
	klog.V(3).Infof("method = %s, url = %s, body = %s", req.Method, req.URL.String(), string(data))
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("Failed to call BIG-IP API: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return err
	}
	klog.V(3).Infof("response: body = %s", string(respBody))
	var response map[string]interface{}
	if err = json.Unmarshal(respBody, &response); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return err
	}
	err = handleResponse(resp.StatusCode, response)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) getF5Resource(url string) (response map[string]interface{}, err error) {
	req, err := http.NewRequest(http.MethodGet, c.host+url, nil)
	if err != nil {
		klog.Errorf("Failed to get BIG-IP resource request: %v", err)
		return
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("Failed to call BIG-IP API: %v", err)
		return
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return
	}
	if err = json.Unmarshal(respBody, &response); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return
	}
	if err = handleResponse(resp.StatusCode, response); err != nil {
		return
	}
	return
}

func (c *Client) postF5Resouce(obj interface{}, url string) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.host+url, bytes.NewBuffer(data))
	if err != nil {
		klog.Errorf("Failed to create AS3 request: %v", err)
		return err
	}
	klog.V(3).Infof("method = %s, url = %s, body = %s", req.Method, req.URL.String(), string(data))
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("Failed to call AS3 API: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return err
	}
	klog.V(3).Infof("response: body = %s", string(respBody))
	var response map[string]interface{}
	if err = json.Unmarshal(respBody, &response); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return err
	}
	err = handleResponse(resp.StatusCode, response)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) storeDisk() error {
	obj := struct {
		Commond string `json:"command"`
	}{
		Commond: "save",
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	url := c.host + "/mgmt/tm/sys/config"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		klog.Errorf("Failed to create BIG-IP resouce request: %v", err)
		return err
	}
	klog.V(3).Infof("method = %s, url = %s, body = %s", req.Method, req.URL.String(), string(data))
	req.SetBasicAuth(c.username, c.password)
	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("Failed to call BIG-IP API: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return err
	}
	klog.V(3).Infof("response: body = %s", string(respBody))
	var response map[string]interface{}
	if err = json.Unmarshal(respBody, &response); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return err
	}
	return handleResponse(resp.StatusCode, response)
}

// get f5 license key
func (c *Client) getF5LicenseKey() (string, error) {
	url := c.host + "/mgmt/tm/sys/license"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		klog.Errorf("Failed to get bigdata license: %v", err)
		return "", err
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.Do(req)
	if err != nil {
		klog.Errorf("Failed to call bigdata API: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return "", err
	}

	if resp.StatusCode != 200 && string(respBody) == "" {
		return "", fmt.Errorf("Failed to get license key")
	}

	type RegistrationKey struct {
		Description string `json:"description"`
	}

	type License struct {
		RegistrationKey RegistrationKey `json:"registrationKey"`
	}

	type NestedStatsEntries struct {
		Entries License `json:"entries"`
	}

	type LicentseNestedStats struct {
		NestedStats NestedStatsEntries `json:"nestedStats"`
	}

	type LicentseZero struct {
		Zero LicentseNestedStats `json:"https://localhost/mgmt/tm/sys/license/0"`
	}

	type BigDataLicense struct {
		Entries LicentseZero `json:"entries"`
	}

	var license BigDataLicense
	if err = json.Unmarshal(respBody, &license); err != nil {
		klog.Errorf("Failed to unmarshal license body: %v", err)
		return "", err
	}
	return license.Entries.Zero.NestedStats.Entries.RegistrationKey.Description, nil
}

// verify license, If err is nil, the verification passes.
func (c *Client) VerifyLicense(license string, key string) error {
	bigDataLicense, err := c.getF5LicenseKey()
	if err != nil {
		return err
	}

	bytesPass, err := base64.StdEncoding.DecodeString(license)
	if err != nil {
		return err
	}

	tpass, err := AesDecrypt(bytesPass, []byte(key))
	if err != nil {
		return err
	}

	if bigDataLicense != string(tpass) {
		return fmt.Errorf("license is not ok")
	}

	return nil
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func AesDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	//AES分组长度为128位，所以blockSize=16，单位字节
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize]) //初始向量的长度必须等于块block的长度16字节
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS5UnPadding(origData)
	return origData, nil
}
