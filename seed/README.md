# Seed data

- `bank_soal.xlsx` — editable source of the question bank (66 questions: 60 site questions + 6 cross-site reserves `X-01..X-06`, bilingual ID/EN). Also holds `sumber` and `status_validasi`, which are **not** stored in the database.
- `questions.json` — generated from the sheet "Bank Soal": `python seed/convert_bank.py` (needs `pip install openpyxl`). This is what the seeder loads.
- `schools.csv` — columns `name,jenjang,city` (`jenjang`/`city` may be empty). Header only for now. "Sekolah lain" and "Umum / General visitor" come from migration 007.

Load into the Docker database: `make db-seed-strict` (requires exactly 66 questions).
