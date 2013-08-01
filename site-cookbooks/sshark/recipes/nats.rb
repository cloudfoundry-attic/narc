rbenv_gem "nats"

runit_service "nats" do
  default_logger true
  options(:user => "root")
end
