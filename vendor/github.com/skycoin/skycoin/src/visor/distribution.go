package visor

import (
	"github.com/skycoin/skycoin/src/coin"
)

const (
	// MaxCoinSupply is the maximum supply of skycoins
	MaxCoinSupply uint64 = 3e8 // 300,000,000 million

	// DistributionAddressesTotal is the number of distribution addresses
	DistributionAddressesTotal uint64 = 100

	// DistributionAddressInitialBalance is the initial balance of each distribution address
	DistributionAddressInitialBalance uint64 = MaxCoinSupply / DistributionAddressesTotal

	// Initial number of unlocked addresses
	InitialUnlockedCount uint64 = 1

	// UnlockAddressRate is the number of addresses to unlock per unlock time interval
	UnlockAddressRate uint64 = 5

	// UnlockTimeInterval is the distribution address unlock time interval, measured in seconds
	// Once the InitialUnlockedCount is exhausted,
	// UnlockAddressRate addresses will be unlocked per UnlockTimeInterval
	UnlockTimeInterval uint64 = 60 * 60 * 24 * 365 // 1 year
)

func init() {
	if MaxCoinSupply%DistributionAddressesTotal != 0 {
		panic("MaxCoinSupply should be perfectly divisible by DistributionAddressesTotal")
	}
}

// GetDistributionAddresses returns a copy of the hardcoded distribution addresses array.
// Each address has 1,000,000 coins. There are 100 addresses.
func GetDistributionAddresses() []string {
	addrs := make([]string, len(distributionAddresses))
	for i := range distributionAddresses {
		addrs[i] = distributionAddresses[i]
	}
	return addrs
}

// GetUnlockedDistributionAddresses returns distribution addresses that are unlocked, i.e. they have spendable outputs
func GetUnlockedDistributionAddresses() []string {
	// The first InitialUnlockedCount (25) addresses are unlocked by default.
	// Subsequent addresses will be unlocked at a rate of UnlockAddressRate (5) per year,
	// after the InitialUnlockedCount (25) addresses have no remaining balance.
	// The unlock timer will be enabled manually once the
	// InitialUnlockedCount (25) addresses are distributed.

	// NOTE: To have automatic unlocking, transaction verification would have
	// to be handled in visor rather than in coin.Transactions.Visor(), because
	// the coin package is agnostic to the state of the blockchain and cannot reference it.
	// Instead of automatic unlocking, we can hardcode the timestamp at which the first 30%
	// is distributed, then compute the unlocked addresses easily here.

	addrs := make([]string, InitialUnlockedCount)
	for i := range distributionAddresses[:InitialUnlockedCount] {
		addrs[i] = distributionAddresses[i]
	}
	return addrs
}

// GetLockedDistributionAddresses returns distribution addresses that are locked, i.e. they have unspendable outputs
func GetLockedDistributionAddresses() []string {
	// TODO -- once we reach 30% distribution, we can hardcode the
	// initial timestamp for releasing more coins
	addrs := make([]string, DistributionAddressesTotal-InitialUnlockedCount)
	for i := range distributionAddresses[InitialUnlockedCount:] {
		addrs[i] = distributionAddresses[InitialUnlockedCount+uint64(i)]
	}

	return addrs
}

// TransactionIsLocked returns true if the transaction spends locked outputs
func TransactionIsLocked(inUxs coin.UxArray) bool {
	lockedAddrs := GetLockedDistributionAddresses()
	lockedAddrsMap := make(map[string]struct{})
	for _, a := range lockedAddrs {
		lockedAddrsMap[a] = struct{}{}
	}

	for _, o := range inUxs {
		uxAddr := o.Body.Address.String()
		if _, ok := lockedAddrsMap[uxAddr]; ok {
			return true
		}
	}

	return false
}

var distributionAddresses = [DistributionAddressesTotal]string{
	"2EvQ2MwPeqXqaHs9qRFzWAoByz4ph64jSKU",
	"2BR9QGdm5hMxRkr4C5M21DXjWvxB26WjHeX",
	"Nnw5SgEKuUmsHGWBSzashbf5D98AUouXUh",
	"YVj3NEsacjsM67iPfyMY59vPuecwnp6QWz",
	"25Dvmd5uBvmZNsTRVDEBsQ6JRVpqqe7gDNe",
	"22oySh75yDv9vtBYHqgfYydEV8atTBwAnpp",
	"3Lrz2bDGcT7DprDeMJZ2guEyivAJ2phvtG",
	"td595rUgWWzMsZcudMnTVYiW6BRvaghPKw",
	"2ADzbt8F186254xMG6DvyMCfbsXdeiLA2qn",
	"NCpnHjMN9ta94SzJLZoFqCq9jpNqsHmeRh",
	"2JkBfkkvypKCsJYRVgMHwuTupf5u6j21onr",
	"2AgYKqcGFru1pkFbheNUzWkvKUKjY859AEJ",
	"2fJ7F7vviEWVRFBRKHGQVN59YiSKS2KPg5X",
	"2HnBLgUiZ3s843ik9RqgnArSNPXdagP99M1",
	"7wGF8xvTyVuKMAEsUWzbNqrmEwp56PoMcG",
	"2HzvsMgGU7Pg3pkg13AFAmt93cw2gEWiPc8",
	"2MURz4XbBRKriEL9UV8ixxZA837Hmpgtaqc",
	"2F3cbdN2tf1nKS6h5nfzUeMnoj7uRkjLy8e",
	"t4hvfvVChVk6qVajCnoqTTGNP4FTftJytu",
	"2bJsY7zjFH6iq6Fs1fnyX2LpzB6rPXVPWXC",
	"2R3MVa4AeY3UoZF2nubYDuxTesU91kiD2jd",
	"joVSFdo4CaA19Y2jbJJZ5BJyitZqBnmX9M",
	"Rw68wydU8cE33YLnNYq2eDksARfChZYFEp",
	"dDuYLwRQ1yvt9iQetEKrUGhPivAxB9n5M1",
	"2jWneyxn84PByzXMRTE3hnCJtNxBEYC2ufv",
	"D9KZjrFN2etU2SHFBA2G7jDSddh6ZCy26M",
	"b8KG4qRuxkHt29mpP5LNUuwQwBZhUG6RJB",
	"2LfmBFMFAsNf2ZDyNEwoH8RWNHmxSbQvqSb",
	"ruKpSMdayPGyFtYG9EH2QTpcrmTvF4EjAV",
	"2iej2x3fEk4sraSXeShVu65KH2NUAvJFg9N",
	"qcu4nqYcX5rGC3EmoXU8q22vDcPUMpPYXY",
	"2YhrGpKUWXfHrtW6c9FKVKqYrxVHVBXGind",
	"2H5mmwbibJz9KxpwRAuZ4ezvTSDL8iNUQGH",
	"zeh9v9afW7a3Ji5e7SGdzb3Sk2xajTXUMf",
	"xV3XQzAR86Y6r7qA49BsFcLxpQ4xhtASq7",
	"EtNKuCeca61htJigtYuove3Yb6H2Scpitk",
	"kfzfkb7TBQLGaBSN4ssjyfMNfscYjdYcy5",
	"27AKWwt4pxtixUM9PAG4J1at7w8oBivJms5",
	"2F5HzZU4RNYZbjmbCCXbDAsHAcqmAMZkpkp",
	"bnEUiH3HVq95pySibLRxHrTLgxQJhsTgRi",
	"7n9m5HZrVDVCNmVYirkG6WB18fHRnF3ZwK",
	"2GpNJcvfoBLTd21oVFvGmvyshmu5GjUDNgA",
	"2LM8uSr35BvyXNihhzTdSNasbYVX4yEGuUi",
	"zhixmLqy3fYcRUAMZZZuABtwVnxwfXriV3",
	"9n5yUAhSwbGZMMhC7KFNCjZnqC918tnjoY",
	"2m6PiUioXSQKAQCqVj7T5fzrVMAi5DX8tU5",
	"2Dspnm7nyGZhtTB5z8imUoQGfgo1SS6EwpC",
	"JHcV8bshqMKatNsBGLqrGWrzH7SDJepcS1",
	"2DDHTwwtBssBeGmoJS9XsVBXnDiwk5HC3tb",
	"2cAg1FEdMnPZH87xUcRtAoppFdRYwQ4YtcC",
	"LPbrt9qoyj8FCGF2dfBb699xyzg5kCc4Q4",
	"QkHTDahAxUh4UDZpTeU2CGkdBrC5QUcPQu",
	"23XWhmcbvDta6CSFC1gh2hiSUbxHjEtKuum",
	"2ceiFAce8qiFo4f6Lnwo7C3UKSUbgmxULPJ",
	"2Ve4PB7skoMyvVZNfnCd5CXkgoocM7i9kHp",
	"CXTjHi3RrUVu6rf3yNiJXbuCeMreSKA1ko",
	"2VyZQNNtdqn8XZzuUsWHzhJyjvzScoPM1sX",
	"2UZsufCFj8njZjXEXUiqeLSf2W5TYc8EfF5",
	"Dnsmrt8PcecpTr6nL8aA4ojmTu48ASU29b",
	"2JGbfKLTkuBqp4qdbnf6ozMwk1UjmEUNS6P",
	"23yiZLFkJpfbgYouDW7xV5ts8VnrLfveGD8",
	"giW9AvmytkeVxu8SPa8twnosu3Rk6rY5b6",
	"21b7EtTzmKaK1axChQDA3JmXpqCJKXbJ9z7",
	"2DYKHug2b52ekqe7Fk3FT3rt5Y72A8vELwk",
	"2BTttXwBVcBv5GvzGiW9V2aBwTfmkoT1DeP",
	"KDNQ85saCEzsAdwZTZqmarcc38MyDJxNbt",
	"23ZepZd1z21F5UiQTAq4y93XCrxRb558PGs",
	"2HKxSby8dchYjfHnMLkjGZCELDhqDs9jGJS",
	"LC3E2iur5MjbN9L6WhxTYW6ApiMU6hCygs",
	"2kg2qEKjhzdpPrGbV7Sp96muaC8VuJmdTRU",
	"237xfxqmsCnRoxRt51YyLe2iNgkXFUfbZB9",
	"2scTNQnfyZaDHPAhJxqMufYNa6pzzgmFJB",
	"B9Whv9d9TGYC7AHi9QJnANa1f2dY3Jny1r",
	"2EnSJzmbNdKNW8BBHavyTkVtyt6Y7j25ETd",
	"MdDzJg7RGqffo87XpmEVvFGPHAnpr9YmeS",
	"6Pc7ibaQ4CHQLH4HygrgRY9dMcGtRVoqSs",
	"2ZHRzCmZvdQV5R1Fi2c2STAo82VkHqgmvuc",
	"2Xay2CH2usdPYRDwoqQusMPxDEAuS76aAm2",
	"dtEeRybvCVVLeXqbmi4tzzzD5AMYFeeH9A",
	"2SpbEq7LzbEFvZFAFqR7fXBH9aibD2B6iQM",
	"CE5aNvp4qcBeHgaSyF5rFh9k6MamdyfHna",
	"2cEnxLn6h2ojHGC5TRS2BSSy7fgfFodi8dA",
	"WnW3cnehTBAsVDZ7nY8sNb2NM6NQabMPpe",
	"2hFKt4uBBpaT2Qt4QDuWi3cX3rAqQKKwtba",
	"6PxJUUfxZCGhMNueFCsPhGeCNHyXmCmPsd",
	"2atTZmiLmu8oxabcHUYFvQ9KcxMSAtxSKnu",
	"CRjdXLQb4CFbXxcEw2ER42Z95EJamjHkeB",
	"21TaWiWTCZBnC5Mhr8FFGkcg37jdjS12GPc",
	"27sZ4KJbJtbhgiBgtzsNknRc7H7h8YwDNq5",
	"w4MD35w8PTeexgQvbDPMpMf1UhZUVGkdhD",
	"A6FFCRPe7BvgE8oy7o5dhnuSVfSo8vtnkB",
	"2UawChW9sj9EEaVyimore9sbov3fRzif66k",
	"8Eb9dhfj6aTJf6os4M7zmaH1Gy95fmufUD",
	"2GW2zRVkxUkyGGxb4jXvA3TJVX13eS4AjTU",
	"FfsRWPhMRoSMmRFcmMb1knQiAC8RZDRNnA",
	"i5fkXenkrwfBhQUpMLYExJt4T8HYm5Swor",
	"2Q2FJJKULZHjbYdP8Nx5C4cBpoX8nw2gh4a",
	"2PWe6GiM3oKExhXsPHSbwnf3A5fffMKPEE7",
	"nP3BsoFkpbQgYHtp8onsQTFh32VkFQWibB",
	"w2xGLPgGgyTkSVnRgXhyiGpugxCFyMAFpc",
}
