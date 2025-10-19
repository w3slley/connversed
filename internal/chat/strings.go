package chat

const (
	WELCOME           = "Hey %s! Welcome to connversed, your TCP chat application accessed via SSH.\n"
	CLIENT_CONNECTED  = "Client %s connected \n"
	USER_JOINED_ROOM  = "User %s joined the room\n"
	JOINED_ROOM       = "Welcome to the room: %s. \n"
	LEFT_ROOM         = "You have left the room %s. Now you are in the lobby!\n"
	USER_LEFT_ROOM    = "%s left the room.\n"
	ROOM_CREATED      = "Room with id %s was created!\n"
	USER_DISCONNECTED = "User with id %s disconnected!\n"
	NOT_IN_ROOM       = "You are not in a room!\n"
	NO_ROOMS          = "There are no rooms.\n"
	NEW_USERNAME      = "Your username was changed to %s.\n"
	COMMAND_NOT_FOUND = "Message does not contain command.\n"
	INVALID_COMMAND   = "Command '%s' is invalid.\n"
	CURRENT_ROOM_ICON = "* "
)
