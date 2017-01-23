package main

type Permissions struct {
	// API key
	Key string
	// the (secret) VK of the entity that created this permission
	VK string
	// set of permissions
	Subscribe SubscribePermission
	Publish   PublishPermission
	Query     QueryPermission
}

type SubscribePermission struct {
	Allowed bool
}
type PublishPermission struct {
	Allowed bool
}
type QueryPermission struct {
	Allowed bool
}
