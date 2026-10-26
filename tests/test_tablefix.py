import subprocess
import sys

import pytest

import tablefix as tf

MD = """| Field | Journals | Magazines |
|:---|---|---:|
| Learning | PSPI, Educational Psychology Review | Behavioral Scientist |
| Habits | Nature Human Behaviour | Harvard Business Review |
"""

# What Apple Notes keeps after an AI box table is pasted: rules after every row, no header.
NOTES_BOX = """ │ Communication & Storytelling │ PNAS, Journal of Communication │ Nautilus, Aeon │
 ├──────────────────────────────┼────────────────────────────────┼────────────────┤
 │ Improving Learning           │ PSPI                           │ American Educator │
 ├──────────────────────────────┼────────────────────────────────┼────────────────┤
 │ Habit Change                 │ Nature                         │ HBR            │
"""


def test_markdown_header_detected():
    header, rows = tf.parse(MD)
    assert header == ["Field", "Journals", "Magazines"]
    assert rows[1] == ["Habits", "Nature Human Behaviour", "Harvard Business Review"]


def test_notes_box_table_has_no_header():
    header, rows = tf.parse(NOTES_BOX)
    assert header is None
    assert rows == [
        ["Communication & Storytelling", "PNAS, Journal of Communication", "Nautilus, Aeon"],
        ["Improving Learning", "PSPI", "American Educator"],
        ["Habit Change", "Nature", "HBR"],
    ]


def test_markdown_rules_after_every_row_mean_no_header():
    text = "| a | b |\n|---|---|\n| c | d |\n|---|---|\n| e | f |\n"
    header, rows = tf.parse(text)
    assert header is None and len(rows) == 3


def test_wrapped_box_round_trip_keeps_header_and_cells():
    header, rows = tf.parse(MD)
    boxed = tf.box(header, rows, limit=12)
    assert "╞" in boxed  # header rule is heavy
    assert tf.parse(boxed) == (header, rows)


def test_ascii_grid_with_equals_header():
    text = "+---+---+\n| A | B |\n+===+===+\n| 1 | 2 |\n+---+---+\n| 3 | 4 |\n+---+---+\n"
    assert tf.parse(text) == (["A", "B"], [["1", "2"], ["3", "4"]])


def test_escaped_pipe_stays_in_cell():
    header, rows = tf.parse("| a \\| b | c |\n|---|---|\n| 1 | 2 |\n")
    assert header == ["a | b", "c"]
    assert "a \\| b" in tf.markdown(header, rows)


def test_short_rows_are_padded():
    header, rows = tf.parse("| a | b | c |\n|---|---|---|\n| 1 |\n")
    assert rows == [["1", "", ""]]


def test_wide_characters_align():
    out = tf.box(None, [["表", "x"], ["ab", "y"]], 0).splitlines()
    assert len({tf.width(line) for line in out}) == 1


def test_list_uses_header_names_and_bold():
    header, rows = tf.parse(MD)
    out = tf.as_list(header, rows, "*")
    assert out.startswith("*Learning*\n• Journals: PSPI")


def test_html_escapes_cells():
    html = tf.html_table(["<b>"], [["a & b"]])
    assert "&lt;b&gt;" in html and "a &amp; b" in html and "<table" in html


def run(*args, stdin):
    return subprocess.run([sys.executable, "-m", "tablefix", *args], input=stdin, capture_output=True, text=True)


def test_cli_no_table_fails_loudly():
    r = run("md", stdin="hello\n")
    assert r.returncode == 1 and "no table rows" in r.stderr


def test_cli_header_count_mismatch_fails():
    r = run("md", "--header", "x,y", stdin=MD)
    assert r.returncode == 1 and "2 names" in r.stderr


@pytest.mark.parametrize("target", ["notes", "slack", "list", "box", "md"])
def test_cli_targets_print_when_piped(target):
    r = run(target, stdin=MD)
    assert r.returncode == 0 and "Learning" in r.stdout
