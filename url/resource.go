package url

import (
	"bytes"
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/storage"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

//Resource represents a URL based resource, with enriched meta info
type Resource struct {
	URL             string   `description:"resource URL or relative or absolute path" required:"true"`//URL of resource
	Credential      string   `description:"credentials file"`//name of credential file or credential key depending on implementation
	ParsedURL       *url.URL `json:"-"`//parsed URL resource
	Cache           string    `description:"local cache path"`//Cache path for the resource, if specified resource will be cached in the specified path
	CacheExpiryMs   int      //CacheExpiryMs expiry time in ms
	Name            string   `description:"name of resource, for instance Docker Resource could use it as docker instance name"`//name of a resource
	modificationTag int64
}



//Clone creates a clone of the resource
func (r *Resource) Clone() *Resource {
	return &Resource{
		Name:          r.Name,
		URL:           r.URL,
		Credential:    r.Credential,
		ParsedURL:     r.ParsedURL,
		Cache:         r.Cache,
		CacheExpiryMs: r.CacheExpiryMs,
	}
}

var defaultSchemePorts = map[string]int{
	"ssh":   22,
	"scp":   22,
	"http":  80,
	"https": 443,
}

//Host returns url's host name with user name if user name is part of url
func (r *Resource) Host() string {
	result := r.ParsedURL.Hostname() + ":" + r.Port()
	if r.ParsedURL.User != nil {
		result = r.ParsedURL.User.Username() + "@" + result
	}
	return result
}

//CredentialURL returns url's with provided credential
func (r *Resource) CredentialURL(username, password string) string {
	var urlCredential = ""
	if username != "" {
		urlCredential = username
		if password != "" {
			urlCredential += ":" + password
		}
		urlCredential += "@"
	}
	result := r.ParsedURL.Scheme + "://" + urlCredential + r.ParsedURL.Hostname() + ":" + r.Port() + r.ParsedURL.Path
	if r.ParsedURL.RawQuery != "" {
		result += "?" + r.ParsedURL.RawQuery
	}

	return result
}

//Path returns url's path  directory, assumption is that directory does not have extension, if path ends with '/' it is being stripped.
func (r *Resource) DirectoryPath() string {
	if r.ParsedURL == nil {
		return ""
	}
	var result = r.ParsedURL.Path

	parent, name := path.Split(result)
	if path.Ext(name) != "" {
		result = parent
	}
	if strings.HasSuffix(result, "/") {
		result = string(result[:len(result)-1])
	}
	return result
}

//Port returns url's port
func (r *Resource) Port() string {
	port := r.ParsedURL.Port()
	if port == "" && r.ParsedURL != nil {
		if value, ok := defaultSchemePorts[r.ParsedURL.Scheme]; ok {
			port = toolbox.AsString(value)
		}
	}
	return port
}

//LoadCredentialload credential, returns username, password. It takes errorIfEmpty flag to return an error if there is issue with credential
func (r *Resource) LoadCredential(errorIfEmpty bool) (string, string, error) {
	if r.Credential == "" {
		if errorIfEmpty {
			return "", "", fmt.Errorf("Credential was empty: %v", r.Credential)
		}
		return "", "", nil
	}
	credential, err := cred.NewConfig(r.Credential)
	if err != nil {
		return "", "", fmt.Errorf("failed to load credential: %v %v", r.Credential, err)
	}
	return credential.Username, credential.Password, nil
}

//Download downloads data from URL, it returns data as []byte, or error, if resource is cacheable it first look into cache
func (r *Resource) Download() ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("Fail to download content on empty resource")
	}

	if r.Cachable() {
		content := r.readFromCache()
		if content != nil {
			return content, nil
		}
	}

	service, err := storage.NewServiceForURL(r.URL, r.Credential)
	if err != nil {
		return nil, err
	}
	object, err := service.StorageObject(r.URL)
	if err != nil {
		return nil, err
	}
	reader, err := service.Download(object)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if r.Cachable() {
		_ = ioutil.WriteFile(r.Cache, content, 0666)
	}
	return content, err
}

//DownloadText returns a text downloaded from url
func (r *Resource) DownloadText() (string, error) {
	var result, err = r.Download()
	if err != nil {
		return "", err
	}
	return string(result), err
}

//Decode decodes url's data into target, it takes decoderFactory which decodes data into target
func (r *Resource) Decode(target interface{}, decoderFactory toolbox.DecoderFactory) error {
	if r == nil {
		return fmt.Errorf("Fail to %T decode on empty resource", decoderFactory)
	}
	if r == nil {
		return fmt.Errorf("Fail to decode %v,  decoderFactory was empty", r.URL, decoderFactory)
	}
	var content, err = r.Download()
	if err != nil {
		return err
	}
	return decoderFactory.Create(bytes.NewReader(content)).Decode(target)
}

//Rename renames URI name of this resource
func (r *Resource) Rename(name string) (err error) {
	var _, currentName = toolbox.URLSplit(r.URL)
	r.URL = strings.Replace(r.URL, currentName, name, 1)
	r.ParsedURL, err = url.Parse(r.URL)
	return err
}

//JSONDecode decodes json resource into target
func (r *Resource) JSONDecode(target interface{}) error {
	return r.Decode(target, toolbox.NewJSONDecoderFactory())
}

//JSONDecode decodes yaml resource into target
func (r *Resource) YAMLDecode(target interface{}) error {
	return r.Decode(target, toolbox.NewYamlDecoderFactory())
}

func (r *Resource) readFromCache() []byte {
	if toolbox.FileExists(r.Cache) {
		info, err := os.Stat(r.Cache)
		var isExpired = false
		if err == nil && r.CacheExpiryMs > 0 {
			elapsed := time.Now().Sub(info.ModTime())
			isExpired = elapsed > time.Second*time.Duration(r.CacheExpiryMs)
		}
		content, err := ioutil.ReadFile(r.Cache)
		if err == nil && !isExpired {
			return content
		}
	}
	return nil
}

//Cachable returns true if resource is cachable
func (r *Resource) Cachable() bool {
	return r.Cache != ""
}

func computeResourceModificationTag(resource *Resource) (int64, error) {
	service, err := storage.NewServiceForURL(resource.URL, resource.Credential)
	if err != nil {
		return 0, err
	}
	object, err := service.StorageObject(resource.URL)
	if err != nil {
		return 0, err
	}
	var fileInfo = object.FileInfo()

	if object.IsContent() {
		return fileInfo.Size() + fileInfo.ModTime().UnixNano(), nil
	}
	var result int64 = 0
	objects, err := service.List(resource.URL)
	if err != nil {
		return 0, err
	}
	for _, object := range objects {
		objectResource := NewResource(object.URL())
		if objectResource.ParsedURL.Path == resource.ParsedURL.Path {
			continue
		}

		modificationTag, err := computeResourceModificationTag(NewResource(object.URL(), resource.Credential))
		if err != nil {
			return 0, err
		}
		result += modificationTag

	}
	return result, nil
}

func (r *Resource) HasChanged() (changed bool, err error) {
	if r.modificationTag == 0 {
		r.modificationTag, err = computeResourceModificationTag(r)
		return false, err
	}
	var recentModificationTag int64
	recentModificationTag, err = computeResourceModificationTag(r)
	if err != nil {
		return false, err
	}
	if recentModificationTag != r.modificationTag {
		changed = true
		r.modificationTag = recentModificationTag
	}
	return changed, err
}

func normalizeURL(URL string) string {
	if strings.Contains(URL, "://") {
		var protoPosition = strings.Index(URL, "://")
		if protoPosition != -1 {
			var urlSuffix = string(URL[protoPosition+3:])
			urlSuffix = strings.Replace(urlSuffix, "//", "/", len(urlSuffix))
			URL = string(URL[:protoPosition+3]) + urlSuffix
		}
		return URL
	}
	if !strings.HasPrefix(URL, "/") {
		currentDirectory, err := os.Getwd()
		if err == nil {
			candidate := path.Join(currentDirectory, URL)
			if toolbox.FileExists(candidate) {
				URL = candidate
			}
		}
	}
	return toolbox.FileSchema + URL
}

//NewResource returns a new resource for provided URL, followed by optional credential, cache and cache expiryMs.
func NewResource(Params ...interface{}) *Resource {
	if len(Params) == 0 {
		return nil
	}
	var URL = toolbox.AsString(Params[0])
	URL = normalizeURL(URL)

	var credential string
	if len(Params) > 1 {
		credential = toolbox.AsString(Params[1])
	}
	var cache string
	if len(Params) > 2 {
		cache = toolbox.AsString(Params[2])
	}
	var cacheExpiryMs int
	if len(Params) > 3 {
		cacheExpiryMs = toolbox.AsInt(Params[3])
	}
	parsedURL, _ := url.Parse(URL)
	return &Resource{
		ParsedURL:     parsedURL,
		URL:           URL,
		Credential:    credential,
		Cache:         cache,
		CacheExpiryMs: cacheExpiryMs,
	}
}
