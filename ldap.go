package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"

	"gopkg.in/ldap.v2"
)

func checkMembership(cfg *LdapConfiguration, username string) (bool, error) {
	urls := make([]string, len(cfg.Urls))
	copy(urls, cfg.Urls)

	var lastErr error

	for len(urls) > 0 {
		i := rand.Intn(len(urls))
		conn, err := ldapConnect(urls[i], cfg.Cacert)
		if err != nil {
			// try next LDAP server
			lastErr = err
			urls = append(urls[:i], urls[i+1:]...)
			continue
		}
		defer conn.Close()

		groups, err := ldapGetGroups(conn, cfg.BaseDn, username)
		if err != nil {
			// try next LDAP server
			lastErr = err
			urls = append(urls[:i], urls[i+1:]...)
			continue
		}

		if len(groups) == 0 {
			// the user is not found or is not a member of any groups, no point
			// in trying further
			return false, nil
		}

		for _, allowed := range cfg.Groups {
			for _, group := range groups {
				if group == allowed {
					// found a match
					return true, nil
				}
			}
		}

		// one of the LDAP servers said the user is not a member of any of the
		// required groups, no point in trying further
		return false, nil
	}

	// all LDAP servers produced errors
	return false, lastErr
}

func ldapConnect(url, cacert string) (*ldap.Conn, error) {
	useTls := strings.HasPrefix(url, "ldaps")

	var addr string
	if useTls {
		addr = strings.TrimPrefix(url, "ldaps://")
	} else {
		addr = strings.TrimPrefix(url, "ldap://")
	}

	if !strings.Contains(addr, ":") {
		if useTls {
			addr += ":636"
		} else {
			addr += ":389"
		}
	}

	if !useTls {
		return ldap.Dial("tcp", addr)
	}

	tlsCfg := tls.Config{InsecureSkipVerify: true}
	if cacert != "" {
		pem, err := ioutil.ReadFile(cacert)
		if err != nil {
			return nil, err
		}

		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(pem)

		tlsCfg = tls.Config{
			ServerName: strings.Split(addr, ":")[0],
			RootCAs:    certPool,
		}
	}

	return ldap.DialTLS("tcp", addr, &tlsCfg)
}

func ldapGetGroups(conn *ldap.Conn, dn, username string) ([]string, error) {
	sr, err := conn.Search(ldap.NewSearchRequest(
		dn,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf("(cn=%s)", username),
		[]string{"member"},
		nil,
	))
	if err != nil {
		return nil, err
	}

	groups := []string{}
	for _, e := range sr.Entries {
		for _, m := range e.GetAttributeValues("member") {
			// cn=group,ou=...
			groups = append(groups, m[3:strings.Index(m, ",")])
		}
	}

	return groups, nil
}
