HASH=$(ipfs add -rQ public/)
echo "https://fn.lc/ipfs/$HASH"
ipfs name publish --key=fn.lc $HASH
