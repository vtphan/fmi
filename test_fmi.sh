# requires correct output existed and stored in queries.out
go run fmi.go --build test.txt
go run fmi.go -i test.txt.index -q queries.txt > /tmp/queries.out
diff -y queries.out /tmp/queries.out