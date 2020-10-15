GB = 1 * 1024 * 1024 # in KB

class Config
  attr_reader :num_hosts

  def initialize
    @num_hosts = 4
  end

  def parse
    if (index = ARGV.index("--esx-count"))
      @num_hosts = ARGV[index + 1].to_i
    end
  end

  def to_h
    {
        'num_hosts' => @num_hosts,
    }
  end

  def to_s
    to_h.to_s
  end
end
