package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

type CreateGroupTask struct {
	Group
	System bool

	result *Group
}

func (t *CreateGroupTask) ID() int {
	if t.result == nil {
		glog.Fatal("Not run: ", t)
	}
	return t.result.ID
}

func (t *CreateGroupTask) Run(b *BuildContext) error {
	{
		header := buildDirectoryTarHeader("etc", 0755)
		err := b.Layer.Mkdirp([]string{"etc"}, header)
		if err != nil {
			return fmt.Errorf("error creating /etc directory: %v", err)
		}
	}

	groups := &GroupFile{}
	if b.Layer.Exists([]string{"etc", "groups"}) {
		f, err := b.Layer.Open([]string{"etc", "groups"})
		if err != nil {
			return fmt.Errorf("error opening /etc/groups: %v", err)
		}

		defer loggedClose(f, "/etc/groups")

		err = groups.ParsePrimary(f)
		if err != nil {
			return fmt.Errorf("error reading /etc/groups: %v", err)
		}
	}
	if b.Layer.Exists([]string{"etc", "gshadow"}) {
		f, err := b.Layer.Open([]string{"etc", "gshadow"})
		if err != nil {
			return fmt.Errorf("error opening /etc/gshadow: %v", err)
		}

		defer loggedClose(f, "/etc/gshadow")

		err = groups.ParseShadow(f)
		if err != nil {
			return fmt.Errorf("error reading /etc/gshadow: %v", err)
		}
	}

	existing := groups.FindGroup(t.Name)

	if existing != nil {
		// TODO: Verify other details?
		t.result = existing
		return nil
	}

	group := &t.Group
	if group.ID == 0 {
		group.ID = groups.AssignID(t.System)
	}

	err := groups.Add(group)
	if err != nil {
		return fmt.Errorf("error adding group: %v", err)
	}

	{
		primary, err := groups.WritePrimary()
		if err != nil {
			return fmt.Errorf("error serializing /etc/groups: %v", err)
		}

		replace := true
		meta := &tar.Header{}
		meta.Mode = 0644
		meta.Size = int64(len(primary))
		err = b.Layer.AddEntry([]string{"etc", "groups"}, NewSliceByteSource(primary, "/etc/groups"), meta, replace)
		if err != nil {
			return fmt.Errorf("error writing /etc/groups: %v", err)
		}
	}

	{
		shadow, err := groups.WriteShadow()
		if err != nil {
			return fmt.Errorf("error serializing /etc/gshadow: %v", err)
		}

		replace := true
		meta := &tar.Header{}
		meta.Mode = 0600
		meta.Size = int64(len(shadow))
		err = b.Layer.AddEntry([]string{"etc", "gshadow"}, NewSliceByteSource(shadow, "/etc/shadow"), meta, replace)
		if err != nil {
			return fmt.Errorf("error writing /etc/gshadow: %v", err)
		}
	}

	t.result = group

	return nil

}

type GroupFile struct {
	Groups []*Group
}

type Group struct {
	Name           string
	Password       string
	ID             int
	Administrators []string
	Users          []string
}

func (g *GroupFile) FindGroup(name string) *Group {
	for _, group := range g.Groups {
		if group.Name == name {
			return group
		}
	}
	return nil
}

func (g *GroupFile) AssignID(system bool) int {
	ids := make(map[int]*Group)
	for _, group := range g.Groups {
		ids[group.ID] = group
	}
	// TODO: Pick correct range based on system
	id := 100
	for {
		existing, _ := ids[id]
		if existing == nil {
			return id
		}
		id++
	}
}

func (g *GroupFile) Add(newGroup *Group) error {
	existing := g.FindGroup(newGroup.Name)
	if existing != nil {
		return fmt.Errorf("duplicate group: %s", newGroup.Name)
	}
	g.Groups = append(g.Groups, newGroup)
	return nil
}

func (g *GroupFile) ParsePrimary(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		tokens := strings.Split(line, ":")
		if len(tokens) != 4 {
			return fmt.Errorf("unable to parse /etc/group line: %s", line)
		}

		group := &Group{}
		group.Name = tokens[0]
		// Password comes from gshadow
		id, err := strconv.Atoi(tokens[2])
		if err != nil {
			return fmt.Errorf("invalid group id in /etc/group line: %s", line)
		}
		group.ID = id
		group.Users = strings.Split(tokens[3], ",")

		g.Groups = append(g.Groups, group)
	}
	if scanner.Err() != nil {
		return fmt.Errorf("error reading /etc/group file: %v", scanner.Err())
	}
	return nil
}

func (g *GroupFile) WritePrimary() ([]byte, error) {
	var w bytes.Buffer
	for _, group := range g.Groups {
		line := fmt.Sprintf("%s:x:%d:%s\n", group.Name, group.ID, strings.Join(group.Users, ","))
		_, err := w.WriteString(line)
		if err != nil {
			return nil, fmt.Errorf("error writing: %v", err)
		}
	}
	return w.Bytes(), nil
}

func (g *GroupFile) ParseShadow(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		tokens := strings.Split(line, ":")
		if len(tokens) != 4 {
			return fmt.Errorf("unable to parse /etc/gshadow line: %s", line)
		}

		name := tokens[0]
		group := g.FindGroup(name)
		if group == nil {
			return fmt.Errorf("group in /etc/gshadow not found in /etc/group: %s", name)
		}

		group.Password = tokens[1]
		group.Administrators = strings.Split(tokens[2], ",")

		// Users should the same with /etc/group
	}

	if scanner.Err() != nil {
		return fmt.Errorf("error reading /etc/gshadow: %v", scanner.Err())
	}

	return nil
}

func (g *GroupFile) WriteShadow() ([]byte, error) {
	var w bytes.Buffer
	for _, group := range g.Groups {
		password := group.Password
		if password == "" {
			password = "!"
		}
		line := fmt.Sprintf("%s:%s:%s:%s\n", group.Name, password, strings.Join(group.Administrators, ","), strings.Join(group.Users, ","))
		_, err := w.WriteString(line)
		if err != nil {
			return nil, fmt.Errorf("error writing: %v", err)
		}
	}
	return w.Bytes(), nil
}
