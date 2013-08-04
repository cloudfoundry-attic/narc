ROOT_FS_URL = "http://cfstacks.s3.amazonaws.com/lucid64.dev.tgz"

include_recipe "runit"

%w{
  git
  curl
  debootstrap
  iptables
  ruby1.9.3
}.each do |package_name|
  package package_name
end

if ["debian", "ubuntu"].include?(node["platform"])
  if node["kernel"]["release"].end_with? "virtual"
    package "linux-image-extra" do
      package_name "linux-image-extra-#{node['kernel']['release']}"
      action :install
    end
  end
end

package "quota" do
  action :install
end

package "apparmor" do
  action :remove
end

execute "remove remove all remnants of apparmor" do
  command "sudo dpkg --purge apparmor"
end

gem_package "bundler" do
  gem_binary "/usr/bin/gem"
end

git "/opt/warden" do
  repository "git://github.com/cloudfoundry/warden.git"
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

  command "curl -s #{ROOT_FS_URL} | tar zxf -"
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

execute "setup_warden" do
  cwd "/opt/warden/warden"
  command "bundle install && bundle exec rake setup:bin[/opt/warden/config/warden.yml]"
  action :run
end

runit_service "warden" do
  default_logger true
  options(:user => "vagrant")
end
