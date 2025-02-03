Output irule-validator.gif
Require echo

Set Shell "zsh"
Set FontSize 24
Set Width 1200
Set Height 600
Set TypingSpeed 80ms
Set WindowBar RingsRight

Type "irule-validator regex.irule"
Sleep 800ms
Enter

Type "irule-validator bad-rule.irule"
Sleep 800ms
Enter

Type "irule-validator -p bad-rule.irule"
Sleep 800ms
Enter

Type "irule-validator -p bad-rule2.irule"
Sleep 800ms
Enter

Sleep 5s
