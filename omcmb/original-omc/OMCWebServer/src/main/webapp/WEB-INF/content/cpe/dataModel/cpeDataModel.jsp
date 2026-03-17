<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	 i.disabled { opacity: 0.6; }
	.commonBorder2 { border: 1px solid #E9EDF9; }
	.container, .commonHeight { height: 100%; }
	.commonFlex { display: flex; }
	.commonDisplay { display: inline-block; }
	.commonDisplayBlock { display: block; }
	.commonBorder { border: 1px solid #E9E9E9; }
	.commonBackground { background: #F5F7FE; }
	.commonFontSize14, .commonToolBarBox .el-query .advanceQuery .el-icon { font-size: 14px; }
	.commonFontSize12 { font-size: 12px; }
	.commonColor, .commonToolBarBox .el-query .advanceQuery .el-icon:before, .commonIcon:before { color: #7A7992; }
	.commonColor2 {  color: #999999; }
	.commonColor3 {  color: #333333; }
	.commonFontWeight { font-weight: bold; }
	.commonRight30 { margin-right: 30px; }
	.commonBorderRadius { border-radius: 10px;}
	.commonToolBarBox { height: 30px; }
	.commonBottom10 { margin-bottom: 10px; }
	.commonRight10 { padding-right: 10px;}
	.commonLeft20 { padding-left: 20px;}
	.commonContent { justify-content: space-between; }
	.eNBTableTitle { line-height: 30px;}
	.commonToolBarBox .el-query{ right: 56px; }
	.commonToolBarBox .el-query .advanceQuery { height: 26px; border-radius: 8px; }
	.commonToolBarBox .el-query .advanceQuery .el-input.el-input--small{ width: 250px; }
	.commonToolBarBox .el-query .advanceQuery .el-input.el-input--small .el-input__inner { height: 26px; line-height: 26px; }
	.el-table th { background: #F9F9F9; }
	.el-table td, .commonTextColor { color: #666666; }
	.el-table--border th { border-right: 1px solid #E9E9E9; }
	.el-table th>.cell { color: #999999; font-weight: normal; }
	.executeStatusQuery .el-query{ right: 0; }
	.statusSize { font-size: 18px; margin-right: 4px; margin-top: 2px; }
	.el-icon-status-disable:before { color: #B8C3D9; }
	.el-icon-status-enable:before { color: #4ED76E; }
	.statusText { font-size: 12px; color: #666666; }
	.el-progress-bar { width: 85px; }
	.el-progress__text { display: none; }
	.runningStatus {  width: 90px; background: #E4F1FF;  color: #4D84FF; }
	.successStatus {  width: 76px; background: #EEFFF3;  color: #4ED76E;}
	.skpiStatus { width: 90px; background: #FFFDE3;  color: #ffaa00;}
	.failStatus { width: 50px; background: #FEF2F2;  color: #FF6D59; }
	.commonStatus { height: 24px; line-height: 24px; border-radius: 100px; text-align: center; }
	.commonWidth166 .el-input{ width: 150px; }

	.el-switch.is-checked .el-switch__core { border-color: #4D84FF !important; background-color: #4D84FF !important; }
	.addSlide { width: 100% !important; height: 100% !important; border-radius: 10px; }
	.addSlide .el-card__header { height: 40px !important; line-height: 40px !important; }
	.tabTitleIcon { padding: 0 10px 0 0; }
	.tabTitleIcon:before { font-size: 16px; }
	.el-tabs__item.is-active .tabTitleIcon:before { color: #4D84FF; }
	.queryGroup { height: 26px; border-radius: 8px; }
	.queryGroup .el-input { width: 280px; }
	.queryGroup .el-input__inner { width: 280px; height: 26px; line-height: 26px; padding: 0; }
	.queryGroup .el-icon-common-search { font-size: 14px; }
	.queryGroup .el-icon-common-search:before { color: #7A7992; }
	.addBtnStyle { top: 0; margin-right: 8px !important; }
	.leftWarp {flex: 1; height: 100%; position: relative; overflow: hidden; flex-direction: column; border-radius: 10px; }
	.rightBox { flex: 0 322px; height: calc(100% - 2px);  margin-left: 10px; border-radius: 10px; overflow: hidden; }
	.commonFormFotter { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; }
	.commonBackgroundWhite { background: #FFFFFF; }
	.commonTitle { height: 48px; line-height: 48px; }
	.rightHeaderBox .addTitle {  padding: 0; }
	.rightTextBox { padding: 10px 0; }
	.commonBorderBottom { border-bottom: 1px solid #E9EDF9; }
	.commonBorderTop { border-top: 1px solid #E9EDF9; }
	.rightContent { padding: 16px 0 0; }
	.rightContent .el-input{ width: 282px } 
	.width280 .el-input{ width: 280px; }
	.rightContent .el-form-item__label { color: #666666; }
	.rightContent .el-input__inner{ height: 30px; line-height: 30px; }
	.rightContent .el-input-group__append { padding: 0 8px; background-color: #FFFFFF; }
	.rightContent .el-input-group__append .el-icon { font-size: 14px; }
	.rightContent .el-input-group__append .el-icon:before { color: #7A7992; }
	.ipErrorTip { color: #FA5555; font-size: 12px; }
	.resultBox { border-radius: 4px; background-color: #FFFFFF; margin-top: 5px; width: 280px; max-height: 122px; padding: 5px 0; overflow: auto;}
	.resultBox .el-form-item { margin-right: 0 !important;}
	.form-suffix { position: relative; margin: 3px 0 0 20px; padding: 0; width: 240px; border: none; background: #fff; }
	.form-suffix:hover { background: #F4F9FF; border-radius: 100px; }
	.form-suffix .deleteCommon { display: none; position: absolute; right: 0; top: 8px; }
	.form-suffix:hover .deleteCommon { display: inline-block !important; }
	.form-suffix .text { padding-left: 10px; color: #666666; }
	.form-suffix .ipTextWarp { font-size: 14px; }
	.operBtn { width: 26px; height: 26px; border: 1px solid #D7D7E6; border-radius: 8px; margin-top: 1px; }
	.operBtn .el-icon { line-height: 26px; }
	.operBtn .el-icon:before, .importIcon:before { color: #7A7992; }
	.container .group { padding-left: 26px; }
	.group-title { color: #7A7992; }
	.mainContent .el-form-item { margin-bottom: 22px; }
	.mainContent .el-form-item__label { line-height: 28px; } 
	.AddTitle { padding: 0 30px; }
	.closeIconBox { margin-top: 10px; }
	.el-dialog__footer { height: 48px; }
	.el-tooltip__popper.is-dark { margin: 0 30px 0 50px; }
</style> 

<div id="productModelPage" class="container commonFlex" style='position: relative;'>
	<div class='leftWarp'>
		<!-- 操作按钮-->
		<div class="operations addBtn">
			<div class="circleIcon placeholder-bt addBtnStyle" placeholder="<%=rb.getString("XinZeng")%>">		
				<span class="el-icon-circle-add el-icon" @click="addProductModelClick"></span>
			</div>
		</div>
		<el-ctable height='100%' ref="productModelTable" class='commonBorderRadius commonBorder commonBottom10'
			:url="productModelUrl" :pagination="true" :time='6' id='productModellist'
			:query-params="productModelParam"
			:row-key="'id'"
			@row-click="rowClick">
			<div slot="toolbar">
				<div class='commonFlex commonToolBarBox commonContent'>
					<div class='commonFontSize14 commonColor commonFontWeight eNBTableTitle commonLeft20'><%=rb.getString("ChanPinXingHao")%></div>
					<el-query type="normal" @query="productModelQuery" placeholder="<%=rb.getString("ChanPinXingHao") %> / <%=rb.getString("ChanPinMingCheng")%>"></el-query>
				</div>
			</div>
			<el-table-column width="40">
				<template slot-scope="scope">
					<div class="el-icon el-icon-operation-more" @click="optClick(scope.row, event)" v-clickoutside="handerClose"></div>
				</template>
			</el-table-column> 
			<el-table-column label="<%=rb.getString("ChanPinXingHao")%>" prop="productModel" show-overflow-tooltip="true" sortable>
				<template slot-scope="scope">
					<div v-if='scope.row.productModel != null && scope.row.productModel.length > 1' style='overflow: hidden; text-overflow: ellipsis; white-space:nowrap;'>{{scope.row.productModel.join(',')}} (<span style='color: #4d84ff'>{{scope.row.productModel.length}}</span>)</div>
					<div v-else-if='scope.row.productModel != null'>{{scope.row.productModel.join(',')}}</div>
				</template>				
			</el-table-column>
			<el-table-column label="<%=rb.getString("ChanPinMingCheng") %>" prop="productName" show-overflow-tooltip="true" sortable></el-table-column>	
			</el-table-column>
		</el-ctable>
		<el-cmenu ref="productModelMenu" :data="productModelMenus" @click="menClick"></el-cmenu>
	</div>
   	<!-- new product model :disabled="isReadOnly"-->
   	<div class='rightBox commonBorder2 commonBackgroundWhite' style='position: relative;' v-show='addProductModelShow'>
		<div class='commonFlex commonContent commonTitle rightHeaderBox commonBorderBottom' style='padding: 0 20px;'>
			<span class='AddTitle commonTextColor commonFontSize14 commonFontWeight'>{{productModelTitle}}</span>
			<span class='closeIconBox' @click='addProductModelCancel'><i class='el-icon el-icon-circle-close'></i></span>
		</div>
   		<div class='rightContent' style='padding: 26px 20px; height: calc(100% - 100px);overflow-x: scroll;'> 
   			<el-form :model="confirmForm" ref="confirmForm" label-position="top" :rules="formRule">
				<el-form-item style='margin-bottom: 16px;' label='<%=rb.getString("ChanPinXingHao") %>' v-if='curType == "add"'>
					<el-form-item prop='productModelStr'>
						<el-input v-model='confirmForm.productModelStr'>
							 <i slot="append" class="el-icon el-icon-plus" @click="addProductModelBtn"></i>
						</el-input>						
					</el-form-item>
					<div class='commonBorder2 resultBox' v-show='confirmForm.productModelList.length > 0'>
						<el-form-item class='suffixItem' v-for='(domain,index) in confirmForm.productModelList'>
							<div class='form-suffix'>
								<span class='text'>{{domain}}</span>
								<span class='form-bt-remove el-icon el-icon-circle-close ipTextWarp deleteCommon' @click.prevent='removeProductModel(domain)'></span>
							</div>
						</el-form-item>
						<el-form-item prop='itemTest'>
							<el-input v-model='confirmForm.itemTest' v-show=false></el-input>
						</el-form-item>
					</div>
					<p class='ipErrorTip'>{{productModelErrorMessage}}</p>
	            </el-form-item> 
				<el-form-item label="<%=rb.getString("ChanPinXingHaoLieBiao")%>" >
					<el-ctable ref="selectProductModelTable" :data="selectProductModelList" class="commonBorder2" :readonly="isReadOnly"
		            	:pagination="false" :rownumber=false :row-key="'text'" 
		            	@selection-change='productModelBatchSelect' height="200px">	                          
		                <el-table-column type="selection" :reserve-selection="true" v-if="curType =='add'"></el-table-column>
						<el-table-column label="<%=rb.getString("ChanPinXingHao") %>" prop="text"></el-table-column> 	                          
		            </el-ctable>   
		            <p class='ipErrorTip' style='margin-top: 2px;'>{{productModelMessage}}</p>
				</el-form-item>
				<el-form-item label="<%=rb.getString("ChanPinMingCheng") %>" prop="productName" class='inputCommon basicLeftLabel'>
	                <el-input v-model="confirmForm.productName" ></el-input>
	            </el-form-item>
			</el-form>   				   				
   		</div>
		<div class='commonFlex commonBorderTop commonFormFotter'>
			<div style='padding: 10px 30px 0;'>
				<el-button type="primary" size="mini" @click='addProductModelSubmit'><%=rb.getString("QueDing")%></el-button>
				<el-button size="mini" @click='addProductModelCancel'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
   	</div> 
	
	<el-dialog title="<%=rb.getString("QueRen")%>" id="delDialog" 
		:visible.sync="delDialog" top="30vh" ref="delDialog" 
			width="660" :close-on-click-modal="false"  @close='cancelDel' append-to-body>
		<div style="margin-bottom:20px;"><%=rb.getString("ShiFouQueRenShanChuGuanXiBiao")%></div>
		<span class='commonColor2 commonFontSize12'><%=rb.getString("ShanChuZaoChengSheBeiCanShuShiBai")%></span>
		<div slot="footer">
			<div style='float: right;'>
				<el-button type="primary" @click="delSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="cancelDel"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
	</el-dialog>
</div>
<script>

	var productModelPageVue = new Vue({
		el: '#productModelPage',
		data() {
			var validateProductModel = function(rule,value,callback){
				if(value === '' || value === null || value === undefined) {
					callback(new Error("<%=rb.getString("BiTian")%>"));
				}else{
					callback();
				}
			};
			return {
				productModelUrl: '${ctx}/dataModel/cpe/productModelProductNameRela/getList.action',				
				productModelParam: {
					search_text: '',
					timeZone: timeZone,
					like_fields: 'model_name,market_name',
					page: 1,
					rows: 50
				},
				productModelMenus: [],
				//new product model
                addProductModelShow: false,
            	productModelErrorMessage: '',
            	productModelMessage: '',
            	selectProductModelList: [],
				curSelectProductModelData: [],            
				confirmForm:{
					id: '', 
					productModelStr: '',
					productModelList: [],
					itemTest: '',           	
					productModel: '',
					productName: ''
				},
				formRule: {
					productName: [{validator: validateProductModel}],
            	},
				pageSize: 50,
				rowData: [],
				curType: '',
				productModelTitle: '',
				isReadOnly: false,
				delDialog: false,
				curModifyData: []
			}
		},
		watch: {
			curSelectProductModelData(row){
    			if(row.length != 0){
    				this.productModelMessage = '';
    			}    			
    		},
		},
		computed: {

		},
		methods: {		
			init(){
				var vm = this;
				
				vm.commonProductModelList();
			},
			//-----------------------------------------------------add product model
			addProductModelClick(){
				var vm = this;
				vm.curType = 'add';
				vm.isReadOnly = false;
				vm.productModelTitle = '<%=rb.getString("XinJianChanPinXingHao")%>';
				vm.addProductModelShow = true;
				vm.curSelectProductModelData = [];
				vm.confirmForm.productModelList = [];
				vm.commonProductModelList();
				vm.$refs.selectProductModelTable.clearSelection();
				vm.$refs.confirmForm.resetFields();
				vm.productModelMessage = '';
				vm.productModelErrorMessage = '';
			},
			//product Model list 数据
			commonProductModelList(){
				var vm = this;
				setTimeout(function(){
					axios.post("${ctx}/dataModel/cpe/productModelProductNameRela/getProductModelList.action",stringify({
	                    page: 1,
	                    rows: 50
	                })).then(function(res){
						var data = res.data;
						vm.selectProductModelList = data.map(function(item){
				   			if (item){
				   				return {text: item}
				   			}
				   		})
				   		
					});
				},0)
			},
			//手动添加
			addProductModelBtn(){
				var vm = this, productModelText = vm.confirmForm.productModelStr, str = '';
				if(productModelText == ''){
					vm.productModelErrorMessage = '<%=rb.getString("QingShuRuChanPinXingHao")%> ';
				}else{
					if(productModelText){
						str = productModelText;
						if(vm.confirmForm.productModelList.indexOf(str) == -1){
							vm.confirmForm.productModelList.push(str);
							vm.confirmForm.productModelStr = '';
							vm.productModelErrorMessage = '';
							vm.productModelMessage = '';
							vm.$refs.confirmForm.validateField('itemTest');
						}else{
							vm.productModelErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}
					}else{
						vm.productModelErrorMessage = '<%=rb.getString("QingShuRuChanPinXingHao")%>';
					}
				}
			},
			//删除
			removeProductModel(item){
				var vm = this;
				var index = vm.confirmForm.productModelList.indexOf(item);
				if(index !== -1){
					vm.confirmForm.productModelList.splice(index,1)
				}
				vm.productModelErrorMessage = '';
			},
			//添加保存
			addProductModelSubmit(){
        		var vm = this, addflag = false, url = '';
        		
        		if(vm.curType == 'add'){
        			var manualpProductModelList = vm.confirmForm.productModelList;
            		var curModelList = vm.curSelectProductModelData.map(function(item){ return item.text });  
        			
        			if(manualpProductModelList.length == 0 && vm.curSelectProductModelData.length == 0 && vm.curType == "add"){
            			vm.productModelMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
            			addflag = true;
            		}else{        		
            			vm.productModelMessage = '';
            			addflag = false;
            			//数组合并 去重
                		var mergeModelList = [];
                		var mergelist = manualpProductModelList.concat(curModelList);
                		for(var i =0, len = mergelist.length; i<len; i++){
                			if(mergeModelList.indexOf(mergelist[i]) === -1){
                				mergeModelList.push(mergelist[i])
                			}
                		}
            		}
        			var params = {
                   			productModel: mergeModelList,
                   			productName: vm.confirmForm.productName
                   		}
            			url = "${ctx}/dataModel/cpe/productModelProductNameRela/addInfo.action";
            			
        		}else{
        			addflag = false;
        			var params = {
               			productModel: vm.curModifyData,
               			productName: vm.confirmForm.productName,
               			id: vm.rowData.id
               		}
        			url = "${ctx}/dataModel/cpe/productModelProductNameRela/modifyInfo.action";
        		}
        		var saveParams = JSON.stringify(params);
        		vm.$refs.confirmForm.validate(function(valid){
					if(valid && addflag == false){
						axios.post(url,saveParams,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
							var data = response.data;
							var message = '<%=rb.getString("ChengGong")%>';
							if(data["success"]){
								vm.$message({
									message:message,
									type:'success',
								})
                                vm.$refs.productModelTable.refresh();
								vm.addProductModelShow = false;
							}else{
								vm.$message.error(data["message"])
							}
						})
					}
				});
			},
			//添加取消
			addProductModelCancel(){
				var vm = this;
				vm.addProductModelShow = false;
        		vm.curSelectProductModelData = [];
        		vm.$refs.confirmForm.resetFields();
        		vm.confirmForm.productModelList = [];
        		vm.$refs.selectProductModelTable.clearSelection();
			},
			//列表选中
			productModelBatchSelect(selection){
				var vm = this; 
				if(selection.length > 0){
					vm.curSelectProductModelData = selection;
					vm.productModelMessage = '';
				}else{
					vm.curSelectProductModelData = [];
				}
			},
			
			//rowClick
			rowClick(row) {
				this.currentRow = row;
			},
			//list search
			productModelQuery(text) {
				this.productModelParam.search_text = text;
			},
			//-------------------------------------------------------------------table operation
			optClick(row, ev) {
				var vm = this;
				
				vm.rowData = row;
				vm.productModelMenus= [
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:'modify', row: row},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del', row: row}
				];
				
		    	vm.$nextTick(function(){
		    		document.body.click();
					vm.$refs.productModelMenu.show(ev);
		    	});
			},
			// menu click
			menClick(ev) {
				var vm = this,
					codes = {
						modify: vm.modifyModel,
						del: vm.deleteModel
					};

				if(codes[ev.code]){
					codes[ev.code](vm.rowData)
				}
			},
			
			// modify
			modifyModel(row){
				var vm = this;
				
				vm.curType = 'edit';
				vm.isReadOnly = true;
				if(vm.rowData){
					vm.curModifyData = vm.rowData.productModel;
					vm.confirmForm.productName = vm.rowData.productName;
			   		vm.selectProductModelList = vm.rowData.productModel.map(function(item){
			   			if (item){
			   				return {text: item}
			   			}
			   		});
			   		setTimeout(function(){
			   			vm.$refs.selectProductModelTable.toggleAllSelection();
			   		},0);
				}
				
				vm.productModelMessage = '';
				vm.productModelTitle = '<%=rb.getString("XiuGaiChanPinXingHao")%>';
				vm.addProductModelShow = true;
			},
			//delete
			deleteModel(row) {
				var vm = this;
				vm.delDialog = true;
				vm.addProductModelShow = false;
			},	
			delSubmit(){
				var vm = this, 
				params = {
					id: vm.rowData.id
				};
				axios.post("${ctx}/dataModel/cpe/productModelProductNameRela/deleteInfo.action",stringify(params)).then(function(response){
					var data = response.data;
					var message = '<%=rb.getString("ChengGong")%>';
					if(data["success"]){
						vm.$message({
    						message:message,
    						type:'success',
    					})
                        vm.$refs.productModelTable.refresh();
					}else{
						vm.$message.error(data["message"])
					}
				})
				vm.delDialog = false;
			},
			cancelDel(){
				var vm = this;
				vm.delDialog = false;
			},
			// table menu hide
			handerClose() {
				var vm = this;
				vm.$refs.productModelMenu.hide();
			},			
		},
		mounted() {
			this.init();
		}
	});

</script>