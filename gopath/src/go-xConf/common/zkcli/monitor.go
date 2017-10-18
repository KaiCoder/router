package zkcli

// Monitor zookeeper会话状态变化监控
type Monitor interface {
	OnConnected() error
	OnConnecting()
	OnDisconnected()
	OnSessionExpired()
	OnNodeChildrenChanged()
	OnNodeDataChanged(nodepath string)
}
