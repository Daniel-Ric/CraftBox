"""Decode saved WebM via WebCodecs samples independently of HTMLVideo drawing."""
import argparse
from pathlib import Path
from PIL import Image
from playwright.sync_api import sync_playwright

parser = argparse.ArgumentParser()
parser.add_argument('--url', default='http://127.0.0.1:5220')
parser.add_argument('--file', type=Path, default=Path('tmp/cinematic-video-motion/motion.webm'))
args = parser.parse_args()
with sync_playwright() as pw:
    browser = pw.chromium.launch(channel='msedge', headless=True, args=['--use-angle=swiftshader', '--enable-unsafe-swiftshader'])
    page = browser.new_page()
    page.route('**/__probe-shell', lambda r: r.fulfill(content_type='text/html', body='<html><body>Video probe</body></html>'))
    page.route('**/__probe-video', lambda r: r.fulfill(content_type='video/webm', body=args.file.read_bytes()))
    page.goto(args.url + '/__probe-shell')
    samples = page.evaluate('''async()=>{
      const {Input,BufferSource,ALL_FORMATS,VideoSampleSink}=await import('/node_modules/mediabunny/dist/modules/src/index.js');
      const input=new Input({source:new BufferSource(await (await fetch('/__probe-video')).arrayBuffer()),formats:ALL_FORMATS});
      const track=await input.getPrimaryVideoTrack(),sink=new VideoSampleSink(track),results=[];
      for(const time of [0,.125,.208]){
        const sample=await sink.getSample(time),frame=sample.toVideoFrame(),data=new Uint8Array(frame.codedWidth*frame.codedHeight*4);
        await frame.copyTo(data,{format:'RGBA'});
        results.push({time:sample.timestamp,width:frame.codedWidth,height:frame.codedHeight,data:Array.from(data)});
        frame.close();sample.close();
      }
      input.dispose();return results;
    }''')
    for index, sample in enumerate(samples):
        data = bytes(sample.pop('data'))
        brightness = sum(value for i, value in enumerate(data) if i % 4 != 3) / (sample['width'] * sample['height'] * 3)
        Image.frombytes('RGBA', (sample['width'], sample['height']), data).save(args.file.with_name(f'decoded-{index}.png'))
        print('DECODED', sample, 'brightness', brightness, flush=True)
    browser.close()
