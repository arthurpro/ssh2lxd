# ssh2lxd

SSH server for direct access to LXD containers

### CLI usage

```
  -d, --debug           enable debug
  -g, --groups string   user must belong to one of the groups to authenticate (default "wheel,lxd")
  -h, --help            print help
  -l, --listen string   listen on :2222 or 127.0.0.1:2222 (default ":2222")
  -n, --noauth          disable public key auth
  -s, --socket string   LXD socket or use LXD_SOCKET (default "/var/snap/lxd/common/lxd/unix.socket")
```

### SSH Access

```
ssh -A -p 2222 host_user+container_name[+container_user]@lxd_host
```

### Features

- Authentication using host OS ssh keys
- Full support for PTY (terminal) mode and remote command execution
- Support for SCP (SFTP is not supported yet)
- Full Ansible support with fallback to SCP
- SSH Agent forwarding