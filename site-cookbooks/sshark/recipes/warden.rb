include_recipe "runit"
include_recipe "sshark::rbenv"
include_recipe "sshark::gvm" # for curl

%w{ build-essential debootstrap quota iptables }.each { |package_name| package package_name }

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
end

execute "Install Dropbear" do
  # pending; need to use rootfs's glibc
  next

  cwd "/opt/warden/rootfs"

  command <<-CMD
    ROOTFS=$PWD
    curl https://matt.ucc.asn.au/dropbear/releases/dropbear-2013.58.tar.bz2 | tar jxf -
    cd dropbear*
    ./configure && make
    cp dropbear $ROOTFS/usr/bin
    cp dropbearkey $ROOTFS/usr/bin
    rm -rf dropbear*
  CMD

  action :run

  not_if "test -f /opt/warden/rootfs/usr/bin/dropbear"
end

%w(warden.yml).each do |config_file|
  cookbook_file "/opt/warden/config/#{config_file}" do
    owner "vagrant"
  end
end

execute "rbenv rehash"

execute "setup_warden" do
  cwd "/opt/warden/warden"
  command "/opt/rbenv/shims/bundle install && /opt/rbenv/shims/bundle exec rake setup:bin[/opt/warden/config/warden.yml]"
  action :run
end

%w(warden).each do |service_name|
  runit_service service_name do
    default_logger true
    options({:user => "vagrant"})
  end
end
