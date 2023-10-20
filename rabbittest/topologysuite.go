package rabbittest

// TopologySuite uses ChannelSuite to create channel and then creates new topology using this channel.
// It is possible to override created exchange, queue or routing key by overriding SetupTest method:
//
//	func (s *MySuite) SetupTest() {
//		s.TopologySuite.Queue = "other-queue-name"
//		s.TopologySuite.SetupTest()
//	}
type TopologySuite struct {
	ChannelSuite

	Exchange string
	Queue    string
	Key      string
}

// SetupTest creates exchange, queue and binds queue to be consume messages with Key.
// Implements suite.SetupTestSuite.
func (s *TopologySuite) SetupTest() {
	s.ChannelSuite.SetupTest()

	if s.Exchange == "" {
		s.Exchange = "my-Exchange"
	}
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

	if s.Queue == "" {
		s.Queue = "my-Queue"
	}
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

	if s.Key == "" {
		s.Key = "my-Key"
	}
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

// TearDownTest deletes exchange and queue.
// Implements suite.TearDownTestSuite.
func (s *TopologySuite) TearDownTest() {
	if err := s.Channel.ExchangeDelete(s.Exchange, false, false); err != nil {
		s.Fail("delete exchange", s.Exchange, err)
	}

	if _, err := s.Channel.QueueDelete(s.Queue, false, false, false); err != nil {
		s.Fail("delete queue", s.Exchange, err)
	}

	s.ChannelSuite.TearDownTest()
}
