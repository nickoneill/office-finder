# office-finder
### fetch office locations and contact info for US representatives via openai

Keeping updated lists of office locations and phone numbers for US representatives is a difficult job. A large portion of the US House changes every two years, and offices are usually added well after a representative gets their first official website on house.gov, and that's aside from the normal operational changes for existing House or Senate members as offices move around or change phone numbers.

In the past we've relied on humans to update these numbers, either through trial and error (disconnected numbers are often reported to [5 Calls](https://5calls.org)) or attempts to automate human discovery via systems like [mechanical turk](https://github.com/TheWalkers/congress-turk).

Large language models give us a compelling tool to gather this information quickly *and* accurately. By asking a generally trained language model to extract addresses and phone numbers from websites, we can recheck these websites frequently and maintain a more up-to-date list of office information. The process is relatively inexpensive, it costs about $0.20 in credits to run this from scratch on all websites using the GPT 4o mini model.

This tool is designed to keep an updated list in json format at `offices.json` for easily diffing between runs but more importantly to contribute the data back to the [united-states/congress-legislators](https://github.com/unitedstates/congress-legislators/blob/main/legislators-district-offices.yaml) repo via the included `upstreamChanges` command.

### setup
* install `go` on your machine
* copy `.env.example` to `.env` and replace your OpenAI API key in the file.

### usage
* run `go run . scrape` to check all representative websites for office information. This will overwrite the `offices.json` file in the root so you can easily see the diffs for what has changed.
* run `go run . validate` to confirm that every representative in the `united-states/congress-legislator` list has offices in the local file.
* run `go run . upstreamChanges` to generate a new `legislators-district-offices.yaml` with the new office changes applied. You can then create a PR in `united-states/congress-legislator` with the changed file for inclusion there.