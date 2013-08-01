include_recipe "runit"
include_recipe "sshark::rbenv"

%w{build-essential curl debootstrap quota iptables}.each do |package_name|
  package package_name
end

rbenv_gem "bundler"

git "/opt/warden" do
  repository "git://github.com/cloudfoundry/warden.git"
  revision "9712451911c7a0fad149f83895169a4062c47fc3" #"2ab01c5fed198ee451837b062f0e02e783519289"
  action :sync
end

%w(config rootfs containers stemcells).each do |dir|
  directory "/opt/warden/#{dir}" do
    owner "vagrant"
    mode 0755
    action :create
  end
end

execute "Install RootFS" do
  cwd "/opt/warden/rootfs"

  command "curl http://cfstacks.s3.amazonaws.com/lucid64.dev.tgz | tar zxf -"
  action :run

  not_if "test -d /opt/warden/rootfs/usr"
end

execute "Install Dropbear" do
  cwd "/opt/warden/rootfs"

  command <<-CMD
    set -e

    curl https://matt.ucc.asn.au/dropbear/releases/dropbear-2013.58.tar.bz2 | tar jxf -
    mv dropbear-* dropbear_build

    echo "cd dropbear_build && ./configure && make" > install-dropbear
    chmod +x install-dropbear
    chroot /opt/warden/rootfs /install-dropbear

    cp dropbear_build/dropbear usr/bin
    cp dropbear_build/dropbearkey usr/bin

    rm -rf dropbear_build
  CMD

  action :run

  not_if "test -f /opt/warden/rootfs/usr/bin/dropbear"
end

cookbook_file "/opt/warden/config/warden.yml" do
  owner "vagrant"
end

execute "rbenv rehash"

execute "setup_warden" do
  cwd "/opt/warden/warden"
  command "/opt/rbenv/shims/bundle install && /opt/rbenv/shims/bundle exec rake setup:bin[/opt/warden/config/warden.yml]"
  action :run
end

runit_service "warden" do
  default_logger true
  options(:user => "vagrant")
end
