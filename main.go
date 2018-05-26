package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
)

type config struct {
	kubeconfig string
	namespace  string
}

func main() {
	log.SetOutput(os.Stderr)
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return &schema.Provider{
				Schema: map[string]*schema.Schema{
					"kubeconfig": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
					},
					"namespace": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
					},
				},
				ResourcesMap: map[string]*schema.Resource{
					"k8s_resources": k8sResources(),
				},
				ConfigureFunc: func(d *schema.ResourceData) (interface{}, error) {
					return &config{
						kubeconfig: d.Get("kubeconfig").(string),
						namespace:  d.Get("namespace").(string),
					}, nil
				},
			}
		},
	})
}

const Dir = "dir"

func k8sResources() *schema.Resource {
	return &schema.Resource{
		Create: k8sResourcesCreate,
		Read:   k8sResourcesRead,
		Update: k8sResourcesUpdate,
		Delete: k8sResourcesDelete,

		Schema: map[string]*schema.Schema{
			Dir: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func k8sResourcesCreate(d *schema.ResourceData, m interface{}) error {
	dir := d.Get(Dir).(string)
	cmd := kubectl(m, "apply", "-f", dir)
	if err := run(cmd); err != nil {
		return err
	}
	return setId(d, m)
}

func k8sResourcesUpdate(d *schema.ResourceData, m interface{}) error {
	dir := d.Get(Dir).(string)
	cmd := kubectl(m, "apply", "-f", dir)
	return run(cmd)
}

func k8sResourcesDelete(d *schema.ResourceData, m interface{}) error {
	dir := d.Get(Dir).(string)
	cmd := kubectl(m, "delete", "-f", dir)
	return run(cmd)
}

func k8sResourcesRead(d *schema.ResourceData, m interface{}) error {
	return setId(d, m)
}

func run(cmd *exec.Cmd) error {
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		cmdStr := cmd.Path + " " + strings.Join(cmd.Args, " ")
		if stderr.Len() == 0 {
			return fmt.Errorf("%s: %v", cmdStr, err)
		}
		return fmt.Errorf("%s %v: %s", cmdStr, err, stderr.Bytes())
	}
	return nil
}

func kubectl(m interface{}, args ...string) *exec.Cmd {
	kubeconfig := m.(*config).kubeconfig
	if kubeconfig != "" {
		args = append([]string{"--kubeconfig", kubeconfig}, args...)
	}
	namespace := m.(*config).namespace
	if namespace != "" {
		args = append([]string{"-n", namespace}, args...)
	}
	return exec.Command("kubectl", args...)
}

func setId(d *schema.ResourceData, m interface{}) error {
	ID := d.Id()
	dir := d.Get(Dir).(string)

	hash, err := getDirHash(dir)
	if err != nil {
		return err
	}

	selfLinks, err := getSelfLinks(dir, m)
	if err != nil {
		return err
	}

	finalHash := getStringHash(hash + selfLinks)
	noResources := selfLinks == ""
	changed := ID != "" && ID != finalHash

	if noResources || changed {
		d.SetId("")
	} else {
		d.SetId(finalHash)
	}
	return nil
}

func getSelfLinks(dirPath string, m interface{}) (string, error) {
	cmd := kubectl(m, "get", "--ignore-not-found", "-f", dirPath, "-o", "json")
	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	if err := run(cmd); err != nil {
		return "", err
	}
	out := stdout.Bytes()

	if strings.TrimSpace(stdout.String()) == "" {
		return "", nil
	}

	var data struct {
		Items []struct {
			Metadata struct {
				Selflink string `json:"selflink"`
			} `json:"metadata"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return "", err
	}

	var selfLinks string
	for link := range data.Items {
		selfLinks += data.Items[link].Metadata.Selflink + "|"
	}
	return selfLinks, nil
}

func getDirHash(dirPath string) (string, error) {
	var dirHash string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if dirPath == path {
			return nil
		}
		fileHash, err := getFileHash(path)
		if err != nil {
			return err
		}
		dirHash += fileHash
		return nil
	})
	if err != nil {
		return "", err
	}
	return getStringHash(dirHash), nil
}

func getFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func getStringHash(str string) string {
	hash := sha1.New()
	hash.Write([]byte(str))
	return hex.EncodeToString(hash.Sum(nil))
}
