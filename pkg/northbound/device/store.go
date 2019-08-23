// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package device

import (
	"context"
	"github.com/atomix/atomix-go-client/pkg/client/map"
	"github.com/atomix/atomix-go-client/pkg/client/primitive"
	"github.com/atomix/atomix-go-client/pkg/client/session"
	"github.com/atomix/atomix-go-local/pkg/atomix/local"
	"github.com/atomix/atomix-go-node/pkg/atomix"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/onos-topo/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"time"
)

// NewAtomixStore returns a new persistent Store
func NewAtomixStore() (Store, error) {
	client, err := util.GetAtomixClient()
	if err != nil {
		return nil, err
	}

	group, err := client.GetGroup(context.Background(), util.GetAtomixRaftGroup())
	if err != nil {
		return nil, err
	}

	devices, err := group.GetMap(context.Background(), "devices", session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		devices: devices,
		closer:  devices,
	}, nil
}

// NewLocalStore returns a new local device store
func NewLocalStore() (Store, error) {
	lis := bufconn.Listen(1024 * 1024)
	node := local.NewLocalNode(lis)
	go func() {
		_ = node.Start()
	}()
	name := primitive.Name{
		Namespace: "local",
		Name:      "devices",
	}
	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(context.Background(), "devices", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial devices")
	}

	devices, err := _map.New(context.Background(), name, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		devices: devices,
		closer:  &nodeCloser{node},
	}, nil
}

type nodeCloser struct {
	node *atomix.Node
}

func (c *nodeCloser) Close() error {
	return c.node.Stop()
}

// Store stores topology information
type Store interface {
	io.Closer

	// Load loads a device from the store
	Load(deviceID ID) (*Device, error)

	// Store stores a device in the store
	Store(*Device) error

	// Delete deletes a device from the store
	Delete(*Device) error

	// List streams devices to the given channel
	List(chan<- *Device) error

	// Watch streams device events to the given channel
	Watch(chan<- *Event) error
}

// atomixStore is the device implementation of the Store
type atomixStore struct {
	devices _map.Map
	closer  io.Closer
}

func (s *atomixStore) Load(deviceID ID) (*Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	kv, err := s.devices.Get(ctx, string(deviceID))
	if err != nil {
		return nil, err
	} else if kv == nil {
		return nil, nil
	}
	return decodeDevice(kv.Key, kv.Value, kv.Version)
}

func (s *atomixStore) Store(device *Device) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(device)
	if err != nil {
		return err
	}

	// Put the device in the map using an optimistic lock if this is an update
	var kv *_map.KeyValue
	if device.Revision == 0 {
		kv, err = s.devices.Put(ctx, string(device.ID), bytes)
	} else {
		kv, err = s.devices.Put(ctx, string(device.ID), bytes, _map.WithVersion(int64(device.Revision)))
	}

	if err != nil {
		return err
	}

	// Update the device metadata
	device.Revision = Revision(kv.Version)
	return err
}

func (s *atomixStore) Delete(device *Device) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if device.Revision > 0 {
		_, err := s.devices.Remove(ctx, string(device.ID), _map.WithVersion(int64(device.Revision)))
		return err
	}
	_, err := s.devices.Remove(ctx, string(device.ID))
	return err
}

func (s *atomixStore) List(ch chan<- *Device) error {
	mapCh := make(chan *_map.KeyValue)
	if err := s.devices.Entries(context.Background(), mapCh); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for kv := range mapCh {
			if device, err := decodeDevice(kv.Key, kv.Value, kv.Version); err == nil {
				ch <- device
			}
		}
	}()
	return nil
}

func (s *atomixStore) Watch(ch chan<- *Event) error {
	mapCh := make(chan *_map.MapEvent)
	if err := s.devices.Watch(context.Background(), mapCh, _map.WithReplay()); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if device, err := decodeDevice(event.Key, event.Value, event.Version); err == nil {
				ch <- &Event{
					Type:   EventType(event.Type),
					Device: device,
				}
			}
		}
	}()
	return nil
}

func (s *atomixStore) Close() error {
	return s.closer.Close()
}

func decodeDevice(key string, value []byte, version int64) (*Device, error) {
	device := &Device{}
	if err := proto.Unmarshal(value, device); err != nil {
		return nil, err
	}
	device.ID = ID(key)
	device.Revision = Revision(version)
	return device, nil
}

// EventType provides the type for a device event
type EventType string

const (
	EventNone     EventType = ""
	EventInserted EventType = "inserted"
	EventUpdated  EventType = "updated"
	EventRemoved  EventType = "removed"
)

// Event is a store event for a device
type Event struct {
	Type   EventType
	Device *Device
}