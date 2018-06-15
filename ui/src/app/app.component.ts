import {Component} from '@angular/core';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent {
  title = 'Banzai Cloud Telescopes';
  columnsToDisplay = ['machineType', 'cpu', 'mem'];
  myDataArray = [
    {"type": "m5.large", "cpus": 4, "memory": 8},
    {"type": "m5.xlarge", "cpus": 8, "memory": 16},
  ];
}
