package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/urfave/cli"
)

func doRegister(c *cli.Context) error {
	cfg := &Config{
		Port:          "2222",
		ListenAddress: "127.0.0.1",
		StaticPath:    "/home/gabe/src/bwproxy",
		BOSSWAVEAgent: "",
		UseIPv6:       false,
	}
	if c.NArg() != 2 {
		log.Fatal("Need to specify entity file and permissions JSON file")
	}
	entityfile := c.Args().Get(0)
	permissionsfile := c.Args().Get(1)

	registryPath := cfg.StaticPath + "/.registry.db"
	registry := newRegistry(registryPath, cfg.BOSSWAVEAgent)

	// open the entity, register it to get a client instance,
	// then compute a new API key
	f, err := os.Open(entityfile)
	if err != nil {
		return err
	}
	entitybytes, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	vk, err := registry.addEntityBytes(entitybytes)
	if err != nil {
		return err
	}
	h := sha256.New()
	seed := make([]byte, 16)
	binary.PutVarint(seed, time.Now().UnixNano())
	h.Write(seed)
	key_bytes := h.Sum(nil)
	key := fmt.Sprintf("%x", key_bytes)

	f, err = os.Open(permissionsfile)
	if err != nil {
		return err
	}
	var perms Permissions
	dec := json.NewDecoder(f)
	if err = dec.Decode(&perms); err != nil {
		return err
	}
	perms.VK = vk // add the vk
	log.Warning("add vk", vk)

	// add the permission/api key mapping to the db
	if err = registry.addPermissions(key, perms); err != nil {
		return err
	}

	fmt.Printf("Key is: %s\n", key)
	return nil
}

func runProxy(c *cli.Context) error {
	cfg := &Config{
		Port:          "2222",
		ListenAddress: "127.0.0.1",
		StaticPath:    "/home/gabe/src/bwproxy",
		BOSSWAVEAgent: "",
		UseIPv6:       false,
	}
	startProxyServer(cfg)
	return nil
}
