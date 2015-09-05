package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type CreateUserTask struct {
	User
	CreateHome bool
	System     bool
}

func (t *CreateUserTask) Run(b *BuildContext) error {
	{
		header := buildDirectoryTarHeader("etc", 0755)
		err := b.Layer.Mkdirp([]string{"etc"}, header)
		if err != nil {
			return fmt.Errorf("error creating /etc directory: %v", err)
		}
	}

	users := &UserFile{}
	if b.Layer.Exists([]string{"etc", "passwd"}) {
		f, err := b.Layer.Open([]string{"etc", "passwd"})
		if err != nil {
			return fmt.Errorf("error opening /etc/passwd: %v", err)
		}

		defer loggedClose(f, "/etc/passwd")

		err = users.ParsePrimary(f)
		if err != nil {
			return fmt.Errorf("error reading /etc/passwd: %v", err)
		}
	}
	if b.Layer.Exists([]string{"etc", "shadow"}) {
		f, err := b.Layer.Open([]string{"etc", "shadow"})
		if err != nil {
			return fmt.Errorf("error opening /etc/shadow: %v", err)
		}

		defer loggedClose(f, "/etc/shadow")

		err = users.ParseShadow(f)
		if err != nil {
			return fmt.Errorf("error reading /etc/shadow: %v", err)
		}
	}

	existing := users.FindUser(t.Name)

	if existing != nil {
		// TODO: Verify other details?
		return nil
	}

	user := &t.User
	if user.ID == 0 {
		user.ID = users.AssignID(t.System)
	}
	if user.Password == "" {
		user.Password = "*"
	}
	err := users.Add(user)
	if err != nil {
		return fmt.Errorf("error adding user: %v", err)
	}

	{
		primary, err := users.WritePrimary()
		if err != nil {
			return fmt.Errorf("error serializing /etc/passwd: %v", err)
		}

		replace := true
		meta := &tar.Header{}
		meta.Mode = 0644
		meta.Size = int64(len(primary))
		err = b.Layer.AddEntry([]string{"etc", "passwd"}, NewSliceByteSource(primary, "/etc/passwd"), meta, replace)
		if err != nil {
			return fmt.Errorf("error writing /etc/passwd: %v", err)
		}
	}

	{
		shadow, err := users.WriteShadow()
		if err != nil {
			return fmt.Errorf("error serializing /etc/shadow: %v", err)
		}

		replace := true
		meta := &tar.Header{}
		meta.Mode = 0600
		meta.Size = int64(len(shadow))
		err = b.Layer.AddEntry([]string{"etc", "shadow"}, NewSliceByteSource(shadow, "/etc/shadow"), meta, replace)
		if err != nil {
			return fmt.Errorf("error writing /etc/shadow: %v", err)
		}
	}

	return nil

}

type UserFile struct {
	Users []*User
}

type User struct {
	Name                     string
	Password                 string
	ID                       int
	GroupID                  int
	GECOS                    string
	HomeDir                  string
	Shell                    string
	PasswordLastChange       string
	PasswordMinAge           string
	PasswordMaxAge           string
	PasswordWarningPeriod    string
	PasswordInactivityPeriod string
	AccountExpiration        string
	Reserved                 string
}

func (g *UserFile) FindUser(name string) *User {
	for _, user := range g.Users {
		if user.Name == name {
			return user
		}
	}
	return nil
}

func (g *UserFile) AssignID(system bool) int {
	ids := make(map[int]*User)
	for _, user := range g.Users {
		ids[user.ID] = user
	}

	// TODO: Is this numbering logic right?
	id := 1000
	if system {
		id = 100
	}

	for {
		existing, _ := ids[id]
		if existing == nil {
			return id
		}
		id++
	}
}

func (g *UserFile) Add(newUser *User) error {
	existing := g.FindUser(newUser.Name)
	if existing != nil {
		return fmt.Errorf("duplicate user: %s", newUser.Name)
	}
	g.Users = append(g.Users, newUser)
	return nil
}

func (g *UserFile) ParsePrimary(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		tokens := strings.Split(line, ":")
		if len(tokens) != 7 {
			return fmt.Errorf("unable to parse /etc/passwd line: %s", line)
		}

		user := &User{}
		user.Name = tokens[0]
		// Password comes from shadow
		id, err := strconv.Atoi(tokens[2])
		if err != nil {
			return fmt.Errorf("invalid user id in /etc/passwd line: %s", line)
		}
		user.ID = id
		groupID, err := strconv.Atoi(tokens[3])
		if err != nil {
			return fmt.Errorf("invalid group id in /etc/passwd line: %s", line)
		}
		user.GroupID = groupID

		user.GECOS = tokens[4]
		user.HomeDir = tokens[5]
		user.Shell = tokens[6]

		g.Users = append(g.Users, user)
	}
	if scanner.Err() != nil {
		return fmt.Errorf("error reading /etc/passwd file: %v", scanner.Err())
	}
	return nil
}

func (g *UserFile) WritePrimary() ([]byte, error) {
	var w bytes.Buffer
	for _, user := range g.Users {
		line := fmt.Sprintf("%s:x:%d:%d:%s:%s:%s\n", user.Name, user.ID, user.GroupID, user.GECOS, user.HomeDir, user.Shell)
		_, err := w.WriteString(line)
		if err != nil {
			return nil, fmt.Errorf("error writing: %v", err)
		}
	}
	return w.Bytes(), nil
}

func (g *UserFile) ParseShadow(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		tokens := strings.Split(line, ":")
		if len(tokens) != 9 {
			return fmt.Errorf("unable to parse /etc/shadow line: %s", line)
		}

		name := tokens[0]
		user := g.FindUser(name)
		if user == nil {
			return fmt.Errorf("user in /etc/shadow not found in /etc/passwd: %s", name)
		}

		user.Password = tokens[1]
		user.PasswordLastChange = tokens[2]
		user.PasswordMinAge = tokens[3]
		user.PasswordMaxAge = tokens[4]
		user.PasswordWarningPeriod = tokens[5]
		user.PasswordInactivityPeriod = tokens[6]
		user.AccountExpiration = tokens[7]
		user.Reserved = tokens[8]
	}

	if scanner.Err() != nil {
		return fmt.Errorf("error reading /etc/shadow: %v", scanner.Err())
	}

	return nil
}

func (g *UserFile) WriteShadow() ([]byte, error) {
	var w bytes.Buffer
	for _, user := range g.Users {
		line := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s:%s\n",
			user.Name,
			user.Password,
			user.PasswordLastChange,
			user.PasswordMinAge,
			user.PasswordMaxAge,
			user.PasswordWarningPeriod,
			user.PasswordInactivityPeriod,
			user.AccountExpiration,
			user.Reserved)
		_, err := w.WriteString(line)
		if err != nil {
			return nil, fmt.Errorf("error writing: %v", err)
		}
	}
	return w.Bytes(), nil
}
