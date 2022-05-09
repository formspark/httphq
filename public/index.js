/* Time formatting */

const timeFormatter = new Intl.RelativeTimeFormat("en", {
  numeric: "auto",
});

const TIME_DIVISIONS = [
  { amount: 60, name: "seconds" },
  { amount: 60, name: "minutes" },
  { amount: 24, name: "hours" },
  { amount: 7, name: "days" },
  { amount: 4.34524, name: "weeks" },
  { amount: 12, name: "months" },
  { amount: Number.POSITIVE_INFINITY, name: "years" },
];

function formatTimeAgo(date) {
  let duration = (date - new Date()) / 1000;
  for (let i = 0; i <= TIME_DIVISIONS.length; i++) {
    const division = TIME_DIVISIONS[i];
    if (Math.abs(duration) < division.amount) {
      return timeFormatter.format(Math.round(duration), division.name);
    }
    duration /= division.amount;
  }
}

/* Body formatting */

function formatBody(body) {
  try {
    return JSON.stringify(JSON.parse(body), null, 2);
  } catch {
    return body;
  }
}
