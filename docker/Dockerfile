FROM almalinux/almalinux:8

RUN dnf install epel-release -y && \
  dnf update -y && \
  dnf install mc wget nano patch yum-utils which rpm-build rpmdevtools git createrepo_c -y && \
  dnf install ruby-devel rubygems -y && \
  dnf groupinstall "Development Tools" -y && \
  gem install --no-document fpm

COPY ./rpmmacros /root/.rpmmacros
COPY ./bashrc /root/.bashrc