# requires correct output existed and stored in queries.out
rm -r ref1.fasta.index
rm /tmp/queries.out
go run substr_search.go --build ref1.fasta
go run substr_search.go -i ref1.fasta.index -q q1.txt > /tmp/queries.out
diff -y queries.out /tmp/queries.out
