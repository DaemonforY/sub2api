export default {
  imageToolUses: {
    title: 'Image tool records',
    description: 'Every AI background removal and AI upscaling run from the canvas and what it cost. Subscribers get free runs every day, then each run is charged to the balance; failed runs are free.',
    today: 'Free runs today',
    todayValue: '{left} of {daily} left',
    noSubscription: 'Subscribers get {daily} free runs a day',
    prices: 'Prices',
    priceValue: 'cut-out ¥{removeBg} / image, upscale ¥{upscale} / image',
    monthRuns: 'Runs this month',
    monthCost: 'Charged this month',
    runsValue: '{runs} ({free} free)',
    tools: { remove_bg: 'AI cut-out', upscale: 'AI upscale' },
    allTools: 'All tools',
    columns: { time: 'Time', tool: 'Tool', charge: 'Charge', size: 'Image size', duration: 'Time taken', key: 'API key' },
    free: 'Free',
    empty: 'No records yet: runs of AI cut-out or AI upscale in the canvas image tools show up here with what they cost.',
    openCanvas: 'Open the canvas'
  }
}
