"""Extract document data without evaluating scripts. Run from any directory."""
import hashlib
import json
import re
from html.parser import HTMLParser
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / 'reference/Manatee_PersonalHome_Page&Field.html'
OUT = ROOT / 'web/src/lib/personalhome'


def catalog():
    source = SOURCE.read_text()
    literal = source.split('const pages = ', 1)[1].split('\n];', 1)[0] + '\n]'
    tokens = re.split(r'("(?:\\.|[^"\\])*")', literal)
    for i in range(0, len(tokens), 2):
        tokens[i] = re.sub(r'//[^\n]*', '', tokens[i])
        tokens[i] = re.sub(r'\b([A-Za-z_][A-Za-z0-9_]*)\s*:', r'"\1":', tokens[i])
    # Only JSON literals are accepted; executable expressions fail closed.
    pages = json.loads(''.join(tokens))
    return {'source': SOURCE.name, 'sha256': hashlib.sha256(SOURCE.read_bytes()).hexdigest(), 'pages': pages}


class ReportParser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.table = 0
        self.cell = None
        self.row = []
        self.rows = []

    def handle_starttag(self, tag, attrs):
        if tag == 'table':
            self.table += 1
        if self.table != 1:
            return
        if tag == 'tr':
            self.row = []
        if tag == 'td':
            self.cell = []
        if tag == 'br' and self.cell is not None:
            self.cell.append('\n')

    def handle_data(self, data):
        if self.cell is not None:
            self.cell.append(data)

    def handle_endtag(self, tag):
        if self.table != 1:
            return
        if tag == 'td' and self.cell is not None:
            self.row.append(''.join(self.cell).strip())
            self.cell = None
        if tag == 'tr' and len(self.row) == 8:
            self.rows.append(dict(zip(['number', 'column', 'header', 'status', 'confidence', 'locations', 'candidates', 'note'], self.row)))


def report():
    path = ROOT / 'reference/Manatee FL Select HO Rater Effective 12.1.25 - Schema Validation Report.html'
    parser = ReportParser()
    parser.feed(path.read_text())
    return {'source': path.name, 'sha256': hashlib.sha256(path.read_bytes()).hexdigest(), 'columns': parser.rows}


if __name__ == '__main__':
    OUT.mkdir(parents=True, exist_ok=True)
    for name, data in [('catalog', catalog()), ('report', report())]:
        (OUT / f'{name}.json').write_text(json.dumps(data, indent=2) + '\n')
    rules = json.loads((ROOT / 'internal/rules/rulesets/manatee_fl_select_ho_12_1_25.json').read_text())
    templates = json.loads((ROOT / 'internal/rules/rulesets/manatee_fl_select_ho_12_1_25.templates.json').read_text())
    schema = json.loads((ROOT / 'testdata/describe.json').read_text())
    targets = [(r['target']['variant'], r['target']['field']) for r in rules['rules']]
    targets += [(r['variant'], f) for r in templates['records'] for f in r.get('fields', {})]
    target_schema = {}
    for variant, field in targets:
        target_schema[variant + '.' + field] = next(f for f in schema[variant]['fields'] if f['key'] == field)
    (OUT / 'target-schema.json').write_text(json.dumps(target_schema, indent=2, sort_keys=True) + '\n')
    print('Extracted PersonalHome inventory, mapping report and offline target schema.')
