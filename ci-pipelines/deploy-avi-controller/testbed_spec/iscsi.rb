require_relative './lib/config.rb'

config = Config.new
config.parse

avi_controller_ovf_url = ""

$testbed = proc do |*args|
  static_ip_enabled = args.include?('static_ip_enabled:true')
  args.each do |item|
    parts = item.split(":")
    key = parts.first
    value = parts[1..-1].join(":")
    case key
    when 'avi_controller_ovf_url'
      avi_controller_ovf_url = value
    end
  end

  $stderr.puts
  $stderr.puts "Args:"
  $stderr.puts "  avi_controller_ovf_url: #{avi_controller_ovf_url}"

  default = {
    "name" => "tkg-iscsi-datastore",
    "version" => 3,

    "network" => [
       {
         "name" => "net.0",
         "enableDhcp" => true
       }
    ],

    "esx" => (0...config.num_hosts).map do | idx |
      {
      "name" => "esx.#{idx}",
      "vc" => "vc.0",
      "dc" => "dc0",
      "clusterName" => "cluster0",
      "style" => "fullInstall",
      "cpus" => 16,
      "memory" => 64 * 1024,
      "disk" => [24 * GB, 24 * GB],
      "iScsi" => ["iscsi.0"]
      }
    end,

    "iscsi" => [
      {
        "name" => "iscsi.0",
        "luns" => [1024, 1024],
        "iqnRandom" => "nimbus1",
        'ramBacked' => 2,
        'cpus' => 2,
        'memory' => 4096,
        'memoryReservation' => 4096,
        'nicType' => ['vmxnet3'],
      }
    ],

    'ovfVm' => [].tap do |vms|
      vms.push(
        'name' => 'avi-controller',
        'ovfUrl' => avi_controller_ovf_url,
        'nics' => 2,
        'cpus' => 4,
        'memory' => 8096,
        'nicType' => ['vmxnet3', 'vmxnet3'],
        'network' => ['force_public', 'nsx::net.0']
      )
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
