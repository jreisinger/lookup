Lookup FQDN at many [public DNS servers](https://public-dns.info/nameservers.txt).

```
$ go install

$ lookup golang.org 2> /dev/null
lookup at 195.238.40.45   OK
lookup at 109.228.9.40    OK
lookup at 177.86.233.170  OK
<...SNIP...>
failed response from 0 out of 1431 nameservers (0.00%)

$ echo $?
0
```