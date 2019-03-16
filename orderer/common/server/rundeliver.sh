

echo "Deliver" 
for 
do value in {1..5}
BENCHMARK=true go test -run TestOrdererBenchmarkKafkaDeliver/1ch/100000tx/$1kb/50bc/0dc/1ord
done 
