package main

import (
	"fmt"
	"github.com/TensShinet/IslandImage/registry"
	"github.com/davecgh/go-spew/spew"
	"io/ioutil"
)

func main() {
	tempDir, err := ioutil.TempDir("", "islandImage")
	fmt.Println("save dir ", tempDir)
	reg, err := registry.New(registry.Config{
		ImageName: "busybox",
		SaveDir:   tempDir,
	})
	if err != nil {
		fmt.Println("err ", err)
		return
	}
	m, err := reg.GetManifest("")
	spew.Dump(m)
	if err != nil {
		fmt.Println("err ", err)
		return
	}

	if _, err := reg.GetConfig(m); err != nil {
		fmt.Println("err ", err)
		return
	}

	if err := reg.GetLayers(m.Layers); err != nil {
		fmt.Println("err ", err)
	}

}
