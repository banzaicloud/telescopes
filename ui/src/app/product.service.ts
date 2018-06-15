import {Injectable} from '@angular/core';
import {Product} from './product';
import {Observable} from 'rxjs';
import {HttpClient} from '@angular/common/http';

export const PRODUCTS: Product[] = [
  {type: "m5.large", cpus: 4, mem: 8,},
  {type: "m5.xlarge", cpus: 8, mem: 16},
]

@Injectable({
  providedIn: 'root'
})
export class ProductService {

  private productsUrl = 'api/v1/products/ec2/eu-west-1';

  constructor(private http: HttpClient,) {
  }

  getProducts(): Observable<Product[]> {
    return this.http.get<Product[]>(this.productsUrl)
  }
}
