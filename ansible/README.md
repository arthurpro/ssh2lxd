```
ansible-playbook test-ansible.yml -i test-inventory.txt -v
```

ssh config

```
Host lxd2
  Hostname localhost
  Port 2222
  ForwardAgent no
  ProxyJump shuttle2

Host shuttle2
  User root
```