import {Component, OnInit} from '@angular/core';
import {ProductService} from '../product.service';
import {DisplayedProduct, Region} from "../product";
import {Observable} from "rxjs/index";
import {MatTableDataSource} from "@angular/material";

@Component({
  selector: 'app-products',
  templateUrl: './products.component.html',
  styleUrls: ['./products.component.scss'],
})
export class ProductsComponent implements OnInit {

  columnsToDisplay = ['machineType', 'cpu', 'mem', 'regularPrice'];

  regions: Region[];
  provider: string = "ec2";
  region: string;
  products: MatTableDataSource<DisplayedProduct>;

  constructor(private productService: ProductService) {
  }

  ngOnInit() {
    this.updateProducts()
  }

  getRegions(): Observable<Region[]> {
    return new Observable(observer => {
      this.productService.getRegions(this.provider)
        .subscribe(regions => {
          this.regions = regions;
          this.region = regions[0].id;
          observer.next(regions);
        });
    });
  }

  getProducts(): void {
    this.productService.getProducts(this.provider, this.region)
      .subscribe(products => this.products = new MatTableDataSource<DisplayedProduct>(products));
  }

  updateProducts(): void {
    this.getRegions().subscribe(() => {
      this.getProducts()
    })
  }

  applyFilter(filterValue: string) {
    filterValue = filterValue.trim();
    filterValue = filterValue.toLowerCase();
    this.products.filter = filterValue;
  }
}
