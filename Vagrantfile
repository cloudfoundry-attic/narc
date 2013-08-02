Vagrant.configure("2") do |config|
  config.vm.hostname = "sshark"

  config.vm.box = "sshark-precise64"
  config.vm.box_url = "http://files.vagrantup.com/precise64.box"

  config.vm.synced_folder ENV["GOPATH"], "/workspace"

  config.vm.provider :virtualbox do |v, override|
    v.customize ["modifyvm", :id, "--memory", 3*1024]
    v.customize ["modifyvm", :id, "--cpus", 4]
  end

  config.vm.provider :vmware_fusion do |v, override|
    override.vm.box_url = "http://files.vagrantup.com/precise64_vmware.box"
    v.vmx["numvcpus"] = "4"
    v.vmx["memsize"] = 3 * 1024
  end

  config.vm.network :private_network, ip: "192.168.50.4"
  # config.vm.provision :shell,       :path => "scripts/virtualbox_lucid_customize.sh"

  config.vm.provision :shell, :inline => "gem install chef --version 10.26.0 --no-rdoc --no-ri --conservative"

  config.vm.provision :chef_solo do |chef|
    chef.cookbooks_path = ["cookbooks", "site-cookbooks"]

    chef.add_recipe "sshark::apt-update"
    chef.add_recipe "build-essential::default"
    chef.add_recipe "sshark::warden"
    chef.add_recipe "sshark::nats"
    chef.add_recipe "sshark::gvm"
  end
end
