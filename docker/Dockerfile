#FROM almalinux/almalinux:8
FROM arm64v8/centos:8

RUN dnf install epel-release -y && \
  dnf update -y && \
  dnf install centos-release-stream -y && \
  dnf swap centos-{linux,stream}-repos -y && \
  dnf distro-sync -y && \
  dnf install mc wget nano patch yum-utils which rpm-build rpmdevtools dpkg git createrepo_c upx -y && \
  dnf install golang-bin -y && \
  dnf install ruby-devel rubygems -y && \
  dnf groupinstall "Development Tools" -y && \
  gem install --no-document fpm

COPY ./rpmmacros /root/.rpmmacros
COPY ./bashrc /root/.bashrc