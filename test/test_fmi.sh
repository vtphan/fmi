# requires correct output existed and stored in queries.out
rm -r test.txt.index
rm /tmp/queries.out
go run substr_search.go --build test.txt
go run substr_search.go -i test.txt.index -q queries.txt > /tmp/queries.out
diff -y queries.out /tmp/queries.out
