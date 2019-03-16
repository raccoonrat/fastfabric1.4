
echo "Broadcast.."
for value in {1..5}
do
BENCHMARK=true go test -run TestOrdererBenchmarkKafkaBroadcast/1ch/10000tx/$1kb/300bc/0dc/1ord
done

