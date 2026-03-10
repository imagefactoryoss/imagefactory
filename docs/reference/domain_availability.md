# Domain Availability Helper

Some of the names we consider for the public-facing site (e.g. `imgfcy`,
`pixforge`, etc.) are short enough that a quick automated check is useful before
purchasing a TLD.

To avoid repeated manual lookups, we've added a simple Python utility that
produces a list of candidate domains and performs a DNS resolution check plus a
WHOIS query when the `python-whois` package is installed.

## Usage

1. Make sure you have Python 3 available on your path.  Only a small
   dependency is needed:

   ```bash
   pip install python-whois  # optional, helps determine "registered" status
   ```

2. Run the helper script from the workspace root:

   ```bash
   cd /path/to/image-factory
   python scripts/check_domains.py
   ```

   By default the script will probe a handful of bases and the `.com`, `.io`,
   and `.app` TLDs.  You can provide your own list on the command line or via a
   file:

   ```bash
   python scripts/check_domains.py imagefactory imgfactory
   python scripts/check_domains.py --tlds com,net,dev --base-list mynames.txt
   ```

3. Inspect the output.  A domain is likely available if it does *not* resolve
   and the WHOIS status reports `available`.  This is only a heuristic –
   always double check with a registrar before registering.

4. Feel free to add to `DEFAULT_BASES` in the script or supply your own
   generation logic if you want more ideas.

---

The file lives under `docs/reference` so it can be referenced by contributors
when they're picking a name for the project or configuring `IF_REGISTRY_DOMAIN`.
