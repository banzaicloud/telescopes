import {Injectable} from '@angular/core';
import {Product, Products} from './product';
import {Observable} from 'rxjs';
import {map} from 'rxjs/operators';
import {HttpClient} from '@angular/common/http';

export const PRODUCTS: Product[] = [
  {type: "m5.large", cpusPerVm: 4, memPerVm: 8,},
  {type: "m5.xlarge", cpusPerVm: 8, memPerVm: 16},
]

@Injectable({
  providedIn: 'root'
})
export class ProductService {

  private productsUrlBase = 'api/v1/products/';

  constructor(private http: HttpClient) {
  }

  getProducts(provider, region): Observable<Product[]> {
    return this.http.get<Products>(this.productsUrlBase + provider + "/" + region).pipe(
      map(res => {
        return res.products
      })
    )
  }
}
