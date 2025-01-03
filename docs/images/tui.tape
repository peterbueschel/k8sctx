Output tui.gif

Set FontSize 20
Set Width 750
Set Height 400
Set Padding 2
Set Margin 2
Set Framerate 10
Set PlaybackSpeed 1.2
Set Theme "Aardvark Blue"
# setup

Type "# I. Run `ktx` to start the TUI in Filter mode."
Enter
Sleep 2

Type@300ms "ktx"
Sleep 2
Enter

Sleep 2
Type "Start typing and hit Enter to switch to the Select mode"
Sleep 2

Hide
Backspace 55
Show

Type "us-east-1"
Sleep 2
Enter
Sleep 2

Down@500ms 2
Sleep 2

Type "/"
Hide
Backspace 9
Show

Type "In the Select mode press the `/` key to enter the Filter mode"
Sleep 2

Hide
Backspace 61
Show

Type "Leave the Filter mode with the Escape key"
Sleep 2
Escape
Sleep 2
Down@500ms 2
Sleep 2
Enter
Sleep 5
