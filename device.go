package main

import (
	"strconv"
	"time"

	lxd "github.com/lxc/lxd/client"
	log "github.com/sirupsen/logrus"

)

func addProxySocketDevice(server lxd.InstanceServer, container string, socket string) (string, string) {
	instance, etag, err := server.GetInstance(container)
	if err != nil {
		log.Errorf("server.GetInstance error: %v", err)
		return "", ""
	}

	deviceName := "ssh-auth-" + strconv.FormatInt(time.Now().UnixNano(), 16)
	listenSocket := "/tmp/" + deviceName + ".sock"

	_, ok := instance.Devices[deviceName]
	if ok {
		log.Errorf("Device %s already exists for %s", deviceName, instance.Name)
		return "", ""
	}

	device := map[string]string{}
	device["type"] = "proxy"
	device["connect"] = "unix:" + socket
	device["listen"] = "unix:" + listenSocket
	device["bind"] = "container"
	device["mode"] = "0666"

	instance.Devices[deviceName] = device
	op, err := server.UpdateInstance(instance.Name, instance.Writable(), etag)
	if err != nil {
		log.Errorf("server.UpdateInstance error: %v", err)
		return "", ""
	}

	err = op.Wait()
	if err != nil {
		log.Errorf("op.Wait error: %v", err)
		return "", ""
	}

	return deviceName, listenSocket
}

func removeProxySocketDevice(server lxd.InstanceServer, container string, deviceName string){
	instance, etag, err := server.GetInstance(container)
	if err != nil {
		log.Errorf("server.GetInstance error: %v", err)
		return
	}

	_, ok := instance.Devices[deviceName]
	if !ok {
		log.Errorf("Device %s does not exists for %s", deviceName, instance.Name)
		return
	}
	delete(instance.Devices, deviceName)

	op, err := server.UpdateInstance(instance.Name, instance.Writable(), etag)
	if err != nil {
		log.Errorf("server.UpdateInstance error: %v", err)
		return
	}

	err = op.Wait()
	if err != nil {
		log.Errorf("op.Wait error: %v", err)
	}
}
