require_relative './lib/config.rb'

config = Config.new
config.parse

$testbed = proc do |*args|
  static_ip_enabled = args.include?('static_ip_enabled:true')

  default = {
    "name" => "tkg-local-datastore",
    "version" => 3,

    "esx" => (0...config.num_hosts).map do | idx |
      {
        "name" => "esx.#{idx}",
        "vc" => "vc.0",
        "dc" => "dc0",
        "clusterName" => "cluster0",
        "style" => "fullInstall",
        "cpus" => 16,
        "memory" => 64 * 1024,
        "disk" => [1024 * GB, 1024 * GB] ,
      }
    end,
 
    "vcs" => [
      {
        "name" => "vc.0",
        "type" => "vcva",
        "dcName" => ["dc0"],
        "clusters" => [
          {
            "name" => "cluster0",
            "dc" => "dc0",
            "enableDrs" => true,
            "enableHA" => true,
          },
        ]
      }
    ],
  }

  if static_ip_enabled
    default.merge!(
      'worker' => [].tap do |worker|
        worker.push({
          'name' => "worker.0",
          'enableStaticIpService' => true, # turn on static ip server
        })
      end,
    )
  end
  default
end
