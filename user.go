package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	user "github.com/tweekmonster/luser"
)

func getOsUser(u string) (*user.User, error) {
	us, err := user.Lookup(u)
	if err != nil {
		log.Debugf("user.Lookup error: %v", err)
		return nil, err
	}
	return us, nil
}

func getUserAuthKeys(u *user.User) ([][]byte, error) {
	var keys [][]byte

	f, err := os.Open(filepath.Clean(u.HomeDir + "/.ssh/authorized_keys"))
	if err != nil {
		log.Debugf("os.Open error: %v", err)
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		keys = append(keys, s.Bytes())
	}
	return keys, nil
}

func getUserGroups(u *user.User) ([]string, error) {
	groups, err := u.GroupIds()
	if err != nil {
		log.Debugf("user.GroupIds error: %v", err)
		return nil, err
	}
	return groups, nil
}

func parseUser(us string) (string, string, string) {
	var user, container string
	containerUser := "root"
	if strings.Contains(us, "+") {
		u := strings.Split(us, "+")
		user = u[0]
		container = u[1]
		if len(u) > 2 {
			containerUser = u[2]
		}
	} else {
		user = "root"
		container = us
	}
	return user, container, containerUser
}

func getGroupIds(groups []string) []string {
	var new []string
	for _, g := range groups {
		group, err := user.LookupGroup(g)
		if err != nil {
			log.Debugf("user.LookupGroup error: %v", err)
			continue
		}
		new = append(new, string(group.Gid))
	}
	return new
}

func isGroupMatch(a []string, b []string) bool {
	for _, i := range a {
		for _, j := range b {
			if i == j {
				return true
			}
		}
	}
	return false
}
