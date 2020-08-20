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
		Username:  "",
		Password:  "",
		Proxy:     "",
		ImageName: "58.87.123.88:5000/tensshinet/busybox",
		SaveDir:   tempDir,
		UseHttp:   true,
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

	if err := reg.GetConfig(m); err != nil {
		fmt.Println("err ", err)
		return
	}

	if err := reg.GetLayers(m.Layers); err != nil {
		fmt.Println("err ", err)
	}

}
