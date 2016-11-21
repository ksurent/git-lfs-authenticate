package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"strings"

	"gopkg.in/ini.v1"
)

type SshAuthResponse struct {
	Href   string            `json:"href"`
	Header map[string]string `json:"header"`
}

type LdapConfiguration struct {
	Urls   []string
	Groups []string
	BaseDn string
	Cacert string
}

type LfsConfiguration struct {
	Url      string
	User     string
	Password string
}

type Configuration struct {
	Lfs  *LfsConfiguration
	Ldap *LdapConfiguration
}

var errNotAuthorised = errors.New("You're not authorised for this operation")

func main() {
	_, ns, repo, err := figureOutArguments(os.Args[1:])
	if err != nil {
		errOut(err.Error())
	}

	configFile := "/etc/git-lfs-authenticate.conf"
	if configOverride := os.Getenv("GIT_LFS_AUTHENTICATE_CONFIG"); configOverride != "" {
		configFile = configOverride
	}
	cfg, err := readConfig(configFile)
	if err != nil {
		errOutf("Failed to read %q: %s", configFile, err)
	}

	user, err := user.Current()
	if err != nil {
		errOut(err.Error())
	}

	ok, err := checkMembership(cfg.Ldap, user.Username)
	if err != nil {
		errOut(err.Error())
	}
	if !ok {
		errOut(errNotAuthorised.Error())
	}

	u, err := url.Parse(cfg.Lfs.Url)
	if err != nil {
		errOut(err.Error())
	}
	u.Path = "/" + ns + "/" + repo

	res := &SshAuthResponse{
		Href: u.String(),
		Header: map[string]string{
			"Authorization":  "Basic " + httpBasicAuth(cfg.Lfs.User, cfg.Lfs.Password),
			"X-Lfs-From-Ssh": "yes",
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		errOut(err.Error())
	}

	os.Stdout.Write(b)
}

func readConfig(file string) (*Configuration, error) {
	cfg, err := ini.Load(file)
	if err != nil {
		return nil, err
	}

	section := cfg.Section("Ldap")
	ldap := &LdapConfiguration{
		Groups: section.Key("Groups").Strings(","),
		Urls:   section.Key("Urls").Strings(","),
		BaseDn: section.Key("Base").String(),
		Cacert: section.Key("Cacert").String(),
	}

	section = cfg.Section("Lfs")
	lfs := &LfsConfiguration{
		Url:      section.Key("Url").String(),
		User:     section.Key("User").String(),
		Password: section.Key("Password").String(),
	}

	err = cfg.Section("Ldap").MapTo(ldap)
	if err != nil {
		return nil, err
	}

	err = cfg.Section("Lfs").MapTo(lfs)
	if err != nil {
		return nil, err
	}

	return &Configuration{
		Ldap: ldap,
		Lfs:  lfs,
	}, nil
}

func figureOutArguments(args []string) (string, string, string, error) {
	if len(args) < 2 {
		return "", "", "", fmt.Errorf("Expected at least 2 arguments, got %d", len(args))
	}

	if len(args) > 3 {
		// legacy API also passes OID but we don't need it here
		return "", "", "", fmt.Errorf("Expected no more than 3 arguments, got %d", len(args))
	}

	path, operation := args[0], args[1]

	if operation != "upload" && operation != "download" {
		err := fmt.Sprintf("Unknown LFS operation: %q, expected %q or %q", operation, "download", "upload")
		return "", "", "", errors.New(err)
	}

	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		err := fmt.Sprintf("Cannot figure out namespace and repository from path: %q", path)
		return "", "", "", errors.New(err)
	}

	return operation, parts[0], parts[1], nil
}

func httpBasicAuth(user, password string) string {
	pair := user + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(pair))
}

func errOutf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func errOut(msg string) {
	fmt.Fprint(os.Stderr, msg+"\n")
	os.Exit(1)
}
