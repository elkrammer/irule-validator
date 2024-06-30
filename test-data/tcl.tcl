# Define a simple procedure to add two numbers
proc add {x y} {
    return [expr {$x + $y}]
}

# Test variable assignment
set a 10
set b 20

# Test arithmetic operations
set sum [expr {$a + $b}]
set difference [expr {$a - $b}]
set product [expr {$a * $b}]
set quotient [expr {$a / $b}]

# Print results of arithmetic operations
#puts "Sum: $sum"
#puts "Difference: $difference"
#puts "Product: $product"
#puts "Quotient: $quotient"
#
## Test procedure call
#set result [add $a $b]
#puts "Result of add procedure: $result"
#
## Test if-else control structure
#if {$a > $b} {
#    puts "$a is greater than $b"
#} elseif {$a < $b} {
#    puts "$a is less than $b"
#} else {
#    puts "$a is equal to $b"
#}

# Test while loop
#set counter 0
#while {$counter < 5} {
#    puts "Counter: $counter"
#    incr counter
#}

# Test for loop
#for {set i 0} {$i < 5} {incr i} {
#    puts "For loop iteration: $i"
#}

# Test list manipulation
#set myList {1 2 3 4 5}
#puts "List: $myList"
#lappend myList 6
#puts "Appended List: $myList"

# Test string operations
#set myString "Hello, World!"
#puts "String: $myString"
#puts "String length: [string length $myString]"
#puts "Substring: [string range $myString 0 4]"

# Test file operations (uncomment if you want to test file operations)
# set filename "testfile.txt"
# set fileId [open $filename "w"]
# puts $fileId "This is a test file."
# close $fileId

# set fileId [open $filename "r"]
# set fileContent [read $fileId]
# close $fileId
# puts "File Content: $fileContent"
