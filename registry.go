package main

import (
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/immesys/bw2/objects"
	"github.com/pkg/errors"
	bw2 "gopkg.in/immesys/bw2bind.v5"
)

var entityBucket = []byte("entity")
var permissionsBucket = []byte("permissions")

// stores our entities and allows us to pull the BW2Clients using the VKs
type registry struct {
	filename string
	// local file database that stores entities
	db     *bolt.DB
	dbLock sync.Mutex
	// router agent address
	agent string
	// cache of active BW2Clients for each VK
	clients map[string]*bw2.BW2Client
	sync.RWMutex
}

// create a new entity store at the given filename
func newRegistry(filename, agent string) *registry {
	db, err := bolt.Open(filename, 0600, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		log.Fatal(errors.Wrap(err, "Could not open database file"))
	}

	s := &registry{
		db:       db,
		filename: filename,
		agent:    agent,
		clients:  make(map[string]*bw2.BW2Client),
	}

	s.db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucket(entityBucket)
		tx.CreateBucket(permissionsBucket)
		return nil
	})

	s.scanAndLoadVKs()
	s.dbLock.Lock()
	defer s.dbLock.Unlock()
	return s
}

func (s *registry) scanAndLoadVKs() {
	s.Lock()
	defer s.Unlock()
	s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(entityBucket)
		// loop through the bucket and create clients for each of the known keys
		b.ForEach(func(vk, contents []byte) error {
			client := bw2.ConnectOrExit(s.agent)
			vk2, err := client.SetEntity(contents)
			if err != nil {
				log.Error(errors.Wrap(err, "Could not set entity"))
				return nil
			}
			vk_string := base64.URLEncoding.EncodeToString(vk)
			if vk_string != vk2 {
				log.Error(errors.Wrapf(err, "Retrieved vk %s did not match vk from router %s", vk_string, vk2))
				return nil
			}
			s.clients[vk_string] = client
			log.Infof("Loaded vk %s", vk_string)
			return nil
		})
		return nil
	})
}

// Add entity from the given bytes. This will probably be loaded using a file browser
// in the web browser and transmitted
// The entity contents get stored in the entity bucket with the public key (vk) as the key.
// Returns the vk of the key on success
func (s *registry) addEntityBytes(entityContents []byte) (string, error) {
	// read the file to get its contents; this way, we can just store the
	// bytes instead of having to keep the file intact
	fileType := entityContents[0]
	contents := entityContents[1:]

	// parse the contents of the file to extract the vk
	ro, err := objects.NewEntity(int(fileType), contents)
	if err != nil {
		return "", errors.Wrap(err, "Could not parse entity")
	}
	entity := ro.(*objects.Entity)
	vk := entity.GetVK()
	vk_string := base64.URLEncoding.EncodeToString(vk)

	s.dbLock.Lock()
	defer s.dbLock.Unlock()
	err = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(entityBucket)
		return b.Put(vk, contents)
	})

	return vk_string, err
}

func (s *registry) getClientForVK(vk string) *bw2.BW2Client {
	s.RLock()
	defer s.RUnlock()
	return s.clients[vk]
}

func (s *registry) addPermissions(key string, perms Permissions) error {
	s.Lock()
	defer s.Unlock()
	permission_bytes, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(permissionsBucket)
		return b.Put([]byte(key), permission_bytes)
	})
}

func (s *registry) getPermissions(key string) (Permissions, error) {
	var perm Permissions
	return perm, s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(permissionsBucket)
		perm_bytes := b.Get([]byte(key))
		return json.Unmarshal(perm_bytes, &perm)
	})
}
