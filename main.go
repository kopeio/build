package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/glog"
)

type PackageInfo struct {
	Name     string
	Filename string
	SHA256   string
}

func testPackages(callback func(p *PackageInfo)) error {
	path := "examples/Packages.gz"
	packagesgz := NewFileByteSource(path)
	packages := &GZIPByteSource{Inner: packagesgz}

	f, err := packages.Open()
	if err != nil {
		return fmt.Errorf("error opening packages file (%s): %v", path, err)
	}

	defer loggedClose(f, path)

	scanner := bufio.NewScanner(f)

	p := &PackageInfo{}
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if p.Name != "" {
				callback(p)
			}
			p = &PackageInfo{}
		} else {
			colonIndex := strings.Index(line, ": ")
			if colonIndex == -1 {
				glog.Warningf("unable to parse line in packages file: %s", line)
				continue
			}
			key := line[0:colonIndex]
			value := line[colonIndex+2:]

			switch key {
			case "Package":
				p.Name = value
			case "Filename":
				p.Filename = value
			case "SHA256":
				p.SHA256 = value
			}
		}

	}

	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("error reading packages file (%s): %v", path, err)
	}

	return nil
}

func main() {
	/*testPackages(func(p *PackageInfo) {
		if p.Name == "memcached" {
			glog.Info(p)
		}
	})

	os.Exit(1)
	*/

	flag.Set("alsologtostderr", "true")
	flag.Parse()
	buildContext := NewBuildContext()

	baseDir := "examples/memcached"
	files, err := ioutil.ReadDir(baseDir)
	if err != nil {
		glog.Exitf("error listing contents of directory (%s): %v", baseDir, err)
	}

	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, ".deb") {
			t := &AddDebTask{}
			t.Source = NewFileByteSource(baseDir + "/" + file.Name())

			err := t.Run(buildContext)
			if err != nil {
				glog.Exitf("error building image (%s): %v", file.Name(), err)
			}
		} else {
			glog.Infof("Skipping file: %q", name)
		}
	}

	group := &CreateGroupTask{}
	group.Name = "memcache"
	group.System = true
	err = group.Run(buildContext)
	if err != nil {
		glog.Exitf("error building image (%s): %v", group.Name, err)
	}

	user := &CreateUserTask{}
	user.Name = "memcache"
	user.System = true
	user.GroupID = group.ID()
	user.CreateHome = false
	user.GECOS = "Memcached"
	user.Shell = "/bin/false"
	err = user.Run(buildContext)
	if err != nil {
		glog.Exitf("error building image (%s): %v", user.Name, err)
	}

	/*
		     if [ ! -e /etc/memcached.conf ]
															            then
																                mkdir -p /etc
																		            cp /usr/share/memcached/memcached.conf.default /etc/memcached.conf
																			        fi
	*/

	buildContext.Layer.Config.Config.User = "memcache"
	buildContext.Layer.Config.Config.Cmd = []string{"/usr/bin/memcached"}

	buildContext.Layer.DebugDump(os.Stdout, "")

	bundle := NewDockerBundle()
	bundle.Layers = append(bundle.Layers, buildContext.Layer)

	path := "output.tar"
	err = bundle.WriteToFile(path)
	if err != nil {
		glog.Exitf("error writing image: %v", err)
	}

}
