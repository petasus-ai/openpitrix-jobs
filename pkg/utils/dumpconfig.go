package utils

import (
	"io/ioutil"
	"os"
	"regexp"

	"k8s.io/klog"
	"sigs.k8s.io/yaml"
)

var configPath = "/etc/kubesphere/kubesphere.yaml"

// secretKey matches the configuration keys whose values must not reach the log.
var secretKey = regexp.MustCompile(`(?i)(secret|password|passwd|token|accesskey|apikey|privatekey|credential)`)

// DumpConfig logs the configuration file with the values of its secret keys masked.
// Petasus: it used to log the file as it was, S3 keys and all, every five seconds.
func DumpConfig() {
	f, err := os.Open(configPath)
	if err != nil {
		klog.Fatalf("open %s failed, error: %s", configPath, err)
	}

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		klog.Fatalf("read %s failed, error: %s", configPath, err)
	}

	var config interface{}
	if err := yaml.Unmarshal(bytes, &config); err != nil {
		klog.Infof("configmap: (%d bytes, not valid YAML: %s)", len(bytes), err)
		return
	}
	masked, err := yaml.Marshal(mask(config))
	if err != nil {
		klog.Infof("configmap: (%d bytes)", len(bytes))
		return
	}
	klog.Infof("configmap: %s", string(masked))
}

// mask hides every value under a secret key, whatever its type, unless it is a map
// to look into or holds nothing, so that an unset key still shows as unset.
func mask(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			if _, isMap := val.(map[string]interface{}); !isMap && secretKey.MatchString(k) {
				if !empty(val) {
					t[k] = "***"
				}
				continue
			}
			t[k] = mask(val)
		}
	case []interface{}:
		for i := range t {
			t[i] = mask(t[i])
		}
	}
	return v
}

func empty(v interface{}) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	case []interface{}:
		return len(t) == 0
	}
	return false
}
