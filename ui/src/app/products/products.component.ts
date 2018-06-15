import {Component, OnInit} from '@angular/core';
import {ProductService} from '../product.service';
import {Product} from "../product";

@Component({
  selector: 'app-products',
  templateUrl: './products.component.html',
  styleUrls: ['./products.component.scss'],
})
export class ProductsComponent implements OnInit {

  columnsToDisplay = ['machineType', 'cpu', 'mem'];

  provider: string = "gce";
  region: string = "eu-west-1";
  products: Product[];

  constructor(private productService: ProductService) {
  }

  ngOnInit() {
    this.getProducts();
  }

  getProducts(): void {
    this.productService.getProducts(this.provider, "eu-west-1")
      .subscribe(products => this.products = products);
  }

}
