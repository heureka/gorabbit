package rabbittest

// TopologySuite uses ChannelSuite to create channel and then creates new topology using this channel.
type TopologySuite struct {
	ChannelSuite

	Exchange string
	Queue    string
	Key      string
}

func (s *TopologySuite) SetupTest() {
	s.ChannelSuite.SetupSuite()

	s.Exchange = "my-Exchange"
	if err := s.Channel.ExchangeDeclare(
		s.Exchange,
		"topic",
		false,
		false,
		false,
		false,
		nil,
	); err != nil {
		s.FailNow("declare exchange", err)
	}

	s.Queue = "my-Queue"
	if _, err := s.Channel.QueueDeclare(
		s.Queue,
		false,
		false,
		false,
		false,
		nil,
	); err != nil {
		s.FailNow("declare queue", err)
	}

	s.Key = "my-Key"
	if err := s.Channel.QueueBind(
		s.Queue,
		s.Key,
		s.Exchange,
		false,
		nil,
	); err != nil {
		s.FailNow("bind queue", err)
	}
}

func (s *TopologySuite) TearDownTest() {
	if err := s.Channel.ExchangeDelete(s.Exchange, false, false); err != nil {
		s.Fail("delete exchange", s.Exchange, err)
	}

	if _, err := s.Channel.QueueDelete(s.Queue, false, false, false); err != nil {
		s.Fail("delete queue", s.Exchange, err)
	}

	s.ChannelSuite.TearDownTest()
}
