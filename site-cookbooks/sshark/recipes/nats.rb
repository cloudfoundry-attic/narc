gem_package "nats" do
  gem_binary "/usr/bin/gem"
end

runit_service "nats" do
  default_logger true
  options(:user => "root")
end
