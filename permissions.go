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

// returns true if OK, else false
func checkQueryPermissions(perms Permissions, params BWRPCCall) bool {
	return perms.Query.Allowed
}

func checkSubscribePermissions(perms Permissions, params BWRPCCall) bool {
	return perms.Subscribe.Allowed
}

func checkPublishPermissions(perms Permissions, params BWRPCCall) bool {
	return perms.Publish.Allowed
}
