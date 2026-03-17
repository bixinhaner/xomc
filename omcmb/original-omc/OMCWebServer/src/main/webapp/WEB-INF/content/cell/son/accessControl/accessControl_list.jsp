<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.accessDialogStyle .el-icon-circle-info:before{
		color:#CFCFCF;
	}
	.accessDialogStyle .el-input{
		width:585px;
	}
	.el-bulk .el-icon-close{
		font-size:16px !important;
		top:10px !important;
	}
	.deleteSNHostName{
		max-height: 260px; overflow-y: scroll;
	}
	.eleteSnList{
		padding-top: 22px; word-wrap: break-word;
	}
	.deleteHostNameList{
		padding-top:10px;word-wrap: break-word;
	}
</style>
<div id="listDiv" style='display:flex;height:100%;overflow:hidden;position:relative'>
	<div style='flex:1;border-right:1px solid #E9E9E9;position:relative'>
		<div class="circleIcon placeholder-bt CODE_ADVANCE_ACCESS_CONTROL hidden" style="top: 10px;right:55px;" placeholder="<%=rb.getString("TianJia")%>">		
			<span @click='addBlackList' class="el-icon el-icon-circle-add"></span>
		</div>
		<div class="circleIcon placeholder-bt CODE_ADVANCE_ACCESS_CONTROL hidden" style="top: 10px;right:15px;" placeholder="<%=rb.getString("DaoRu")%>">		
			<span @click='importBlackList' class="el-icon el-icon-circle-import"></span>
		</div>
		<el-ctable id="ctableBlack" ref="ctableBlack" :url="blackListUrl" :height="height" @load-success="loadSuccessBlack"
				:row-key="'serialNumber'" :query-params="params_black" pagination="true" :rownumber=true  @selection-change='selectBlack'>
				
			<!-- 模糊查询 -- 接入规则 -->
			<template slot="toolbar">
				<div style='margin-bottom:20px'>
					<span style='font-weight:bold;margin-left:20px;'><%=rb.getString("HeiMingDan")%></span>
					<span style='color:#999;margin-left:20px;'>(<%=rb.getString("HeiMingDanTiShi")%>)</span>
				</div>
				<div class='queryGroup'> 
					<el-input v-model='params_black_form.searchText' @keyup.enter.native="queryBlack" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
					<i @click="queryBlack" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
				</div>
			</template>

			<!-- 主列表 -->
			<el-table-column type="selection" v-if="showCheckFlag"></el-table-column>
			<el-table-column label='' width="30" prop="" class-name="no-text-tips" v-if="showCheckFlag">
				<template slot-scope="scope">
            		<div class="el-icon el-icon-operation-delete" @click="deleteSn('single',scope.row.serialNumber,'','black')" style="cursor: pointer;"></div>
          		</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>'  prop="serialNumber" ></el-table-column>
		</el-ctable>
		<el-cmenu ref="menu_black" :data="menus_black" @click="clickMenuBlack"></el-cmenu>
	</div>
	<div style='flex:1;'>
		<div class="circleIcon placeholder-bt" style="top: 10px;right:37px;" placeholder="<%=rb.getString("GuanBi")%>">		
			<span @click='closeList' class="el-icon el-icon-circle-close"></span>
		</div>
		<div class="circleIcon placeholder-bt CODE_ADVANCE_ACCESS_CONTROL hidden" style="top: 10px;right:115px;" placeholder="<%=rb.getString("TianJia")%>">		
			<span @click='addWhiteList' class="el-icon el-icon-circle-add"></span>
		</div>
		<div class="circleIcon placeholder-bt CODE_ADVANCE_ACCESS_CONTROL hidden" style="top: 10px;right:75px;" placeholder="<%=rb.getString("TianJia")%>">		
			<span @click='importWhiteList' class="el-icon el-icon-circle-import"></span>
		</div>
		<el-ctable id="ctableWhite" ref="ctableWhite" :url="whiteListUrl" :height="height" @load-success="loadSuccessWhite"
				:row-key="'serialNumber'" :query-params="params_white" pagination="true" :rownumber=true  @selection-change='selectWhite'>
				
			<!-- 模糊查询 -- 接入规则 -->
			<template slot="toolbar">
				<div style='margin-bottom:20px'>
					<span style='font-weight:bold;margin-left:20px;'><%=rb.getString("BaiMingDan")%></span>
				</div>
				<div class='queryGroup'> 
					<el-input v-model="params_white_form.searchText" @keyup.enter.native="queryWhite" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
					<i @click="queryWhite" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
				</div>
			</template>

			<!-- 主列表 -->
			<el-table-column type="selection" v-if="showCheckFlag"></el-table-column>
			<el-table-column label='' width="30" prop="" class-name="no-text-tips" v-if="showCheckFlag">
				<template slot-scope="scope">
            		<div class="el-icon el-icon-operation-delete" @click="deleteSn('single',scope.row.serialNumber,scope.row.hostName,'white')" style="cursor: pointer;"></div>
          		</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serialNumber" ></el-table-column>
			<el-table-column label='<%=rb.getString("HostName")%>' prop="hostName"></el-table-column>
		</el-ctable>
		<el-cmenu ref="menu_white" :data="menus_white" @click="clickMenuWhite"></el-cmenu>
	</div>
	<el-dialog class='accessDialogStyle' :title='dialogTitle' width='630px' :visible.sync='listVisible' :append-to-body="true" :close-on-click-modal="false" @close='closeAddList'>
		<el-form ref='addListForm' :rules='addListRules' :model='addListForm' label-position="top">
			<div v-if='handListFlag'>
				<label><%=rb.getString("XiaoZhanBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='addListForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<div v-else>
				<el-form-item label='<%=rb.getString("DaoRuLeiXing")%>'>
					<el-select style='width:585px;' v-model="addListForm.import_type">
						<el-option label='<%=rb.getString("ZhuiJia")%>' value='append'></el-option>
						<el-option label='<%=rb.getString("FuGai")%>' value='cover'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item label='<%=rb.getString("DaoRuWenJian")%>' style='margin-top:20px;' prop="file_path">
					<el-input :disabled="true" style='width:585px;' v-model='addListForm.file_path'>
						<i @click='importFile' slot='suffix' style='display:inline-block;width:28px;height:28px;margin:4px -9px 0 0;' class='el-icon el-icon-operation-import'></i>
					</el-input>
					<p>
						<span class='el-icon el-icon-circle-info' style='font-size:14px;'></span>
						<span style="color:#BBB"><%=rb.getString("DaoRuWenJianTiShi")%></span>
						<span class='el-icon el-icon-common-download'></span>
						<span @click="exportTempList" style='text-decoration:underline;cursor:pointer'><%=rb.getString("DaoChuMuBan")%></span>
					</p>
				</el-form-item>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='saveList' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeAddList'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
	<el-dialog class='resultDialog'  title='<%=rb.getString("DaoRuJieGuo")%>' width='630px' :visible.sync='resultVisible' :append-to-body="true" :close-on-click-modal="false">
		<p><%=rb.getString("ZongLiang")%>：<span>{{count}}</span></p>
		<p><%=rb.getString("DaoRuChengGongShu")%>：<span style='color:#38C846'>{{legalCount}}</span></p>
		<p><%=rb.getString("DaoRuShiBaiShu")%>：<span style='color:#E88282'>{{unlegalCount}}</span></p>
		<div style='width:590px;height:300px;border:1px solid #E4E7EC;margin-top:10px;'>
			<el-ctable ref="ctableResult" :url="resultUrl" :query-params = "params_result" height="100%" pagination="true" :rownumber=true>
				<el-table-column label='Serial Number' prop="serialNumber" width="200"></el-table-column>
				<el-table-column label='Status' prop="status" :formatter="statusFmt"></el-table-column>
			</el-ctable>
		</div>
		<div style='margin-top:45px;'>
			<el-button @click="saveResult" type="primary"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="cancelResult"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	<el-bulk ref="blackBulk" target="ctableBlack" :list="selectBlackList" row-key="serialNumber"  show-prop="serialNumber"
		:message="{title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("XiaoZhanBianMa")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'}">
		<template slot="button">
			<a class="linkbutton linkbutton_trend" @click="deleteSn('all','all','all','black')"><span><%=rb.getString("ShanChu")%></span></a>
		</template>
	</el-bulk>
	<el-bulk ref="whiteBulk" target="ctableWhite" :list="selectWhiteList" row-key="serialNumber"  show-prop="serialNumber"
		:message="{title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("XiaoZhanBianMa")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'}">
		<template slot="button">
			<a class="linkbutton linkbutton_trend" @click="deleteSn('all','all','all','white')"><span><%=rb.getString("ShanChu")%></span></a>
		</template>
	</el-bulk>
	<!-- 删除白名单时提示 删除的基站编码和名称-->
	<el-dialog class='accessDialogStyle' title="<%=rb.getString("QueRen")%>" width='630px' :visible.sync='showDeleteWhiteList' :append-to-body="true" :close-on-click-modal="false" @close='closeWhiteList'>
		<span><%=rb.getString("QueDingShanChuSheBei")%></span>
		<div class="deleteSNHostName">			
			<div class="eleteSnList">
				<span><%=rb.getString("XiaoZhanBianMa")%>: </span>
				<span>{{serialNumberList}}</span>
			</div>
			<div class="deleteHostNameList">
				<span><%=rb.getString("HostName")%>: </span>
				<span>{{hostNameList}}</span>
			</div>
		</div>
		<div style='margin-top:45px;'>
			<el-button @click='saveWhiteList' type="primary"><%=rb.getString("QueDing")%></el-button>
			<el-button @click='closeWhiteList'><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
</div>
<form enctype="multipart/form-data" method="post" id="uploadForm_list" style="display: none;">
	<input name="fileSize"  value="" hidden="true">
    <input name="uploadFile" type="file" id="uploadFileList">
    <input name="operType" value="">
</form>
<script>
	new Vue({
		el:'#listDiv',
		data(){
			var vm = this;
			var validatorNum = (rule,value,callback) => {
				var reg1 = /^(\d|[a-zA-Z]|-){1,30}(;|;\n)?$/;
				var reg2 = /^(\d|[a-zA-Z]|-){1,30}((;|;\n)(\d|[a-zA-Z]|-){1,30}(;|;\n)?)*$/
				if(value == null || value.length == 0){
					callback(new Error('<%=rb.getString("UPSSNBuNengWeiKong")%>'));
				}
				if(!(reg1.test(value) || reg2.test(value))){
					callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
				}
				callback();
			};
			var validateFilePath = (rule,value,callback) => {
				if(value == ""){
					callback(new Error("<%=rb.getString("QingXianXuanZeWenJian")%>"))
				}else if(!fileFormatMatch(value,"xlsx,xls,csv")){
					callback(new Error("<%=rb.getString("DaoRuWenJianGeShi")%>"))
				}else{
					callback();
				}
			};
			return{
				activeName:'black',
				height:'100%',
				blackListUrl:"${ctx}/son/access/queryAccessBlackPageList.action",
				whiteListUrl:"${ctx}/son/access/queryAccessWhitePageList.action",
				params_black:{searchText:''},
				params_black_form:{searchText:''},
				params_white:{searchText:''},
				params_white_form:{searchText:''},
				showAllSelect:false,
				showSelect:false,
				selectNum:0,
				selectBlackList:[],
				selectWhiteList:[],
				showDetail:false,
				listVisible:false,
				addListForm:{
					serialNumber:'',
					import_type:'append',
					file_path:''
				},
				addListRules:{
					serialNumber:[
						{validator:validatorNum,trigger:'blur'}
					],
					file_path:[
						{validator:validateFilePath}
					]
				},
				handListFlag:true,
				resultVisible:false,
				count:'',
				legalCount:'',
				unlegalCount:'',
				params_result:{searchText:''},
				resultUrl:'',
				menus_black:[],
				menus_white:[],
				rowDataBlack:[],
				rowDataWhite:[],
				showCheckFlag:true,
				addType:'',
				dialogTitle:'Add eNB',
				showDeleteWhiteList:false,  
				serialNumberList: '',
				hostNameList: ''
			}
		},
		methods:{
			init(){
				var vm = this;
				if(writableMap["CODE_ADVANCE_ACCESS_CONTROL"]){
					vm.showCheckFlag = true;
				}else{
					vm.showCheckFlag = false;
				}
			},
			queryBlack(){
				Object.assign(this.params_black,this.params_black_form);
			},
			queryWhite(){
				Object.assign(this.params_white,this.params_white_form);
			},
			optClickBlack(row,ev){
				var vm = this;
				vm.rowDataBlack = row;
				vm.menus_black= [
			          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del'}
			    ]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_black.show(ev)
				})
			},
			optClickWhite(row,ev){
				var vm = this;
				vm.rowDataWhite = row;
				vm.menus_white= [
			          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del'}
			    ]
		    	vm.$nextTick(function(){
		    		document.body.click();
  		    		vm.$refs.menu_white.show(ev);
		    	});
			},
			clickMenuBlack(ev){
				var codes = {
		    	    	del:this.deleteSn('single')
		    	    }
	    	    	if(codes[ev.code]){
	    	    		codes[ev.code]()
	    	    	}
			},
			clickMenuWhite(ev){
				var codes = {
		    	    	del:this.deleteSn('single')
		    	    }
	    	    	if(codes[ev.code]){
	    	    		codes[ev.code]()
	    	    	}
			},
			selectBlack(selection){
				this.selectBlackList = selection;
				if(selection.length == 0){
					this.showAllSelect = false;
					this.showSelect = false
				}else{
					this.showAllSelect = true;
					this.showSelect = true
				}
				this.selectNum = selection.length;
			},
			selectWhite(selection){
				this.selectWhiteList = selection;
				if(selection.length == 0){
					this.showAllSelect = false;
					this.showSelect = false
				}else{
					this.showAllSelect = true;
					this.showSelect = true
				}
				this.selectNum = selection.length;
			},
			toDetail(){
				this.showDetail = true;
				
			},
			closeAddList(){
				var vm = this;
				vm.addListForm = {
						import_type:'append',
						file_path:'',
						serialNumber:''
				}
				vm.$nextTick(()=>{
					if(vm.handListFlag){
						vm.$refs.addListForm.clearValidate('serialNumber');
					}else{
						vm.$refs.addListForm.clearValidate('file_path');
					}
				})
				
				this.errorMessage = '';
				$("#uploadForm_list input[name='uploadFile']").val("")
				this.listVisible = false;
			},
			saveList(){
				var vm = this;
				var url = "";
				var type = vm.handListFlag;
				var target;
				if(type){//手动添加
					vm.$refs.addListForm.validate((valid) => {
						if(valid){
							var snArr = vm.addListForm.serialNumber.split(';');
							snArr = snArr.map(function(item){
								return item.replace(/[\r\n]/g,'');
							});
							var serialNumber = snArr.toString().split(",").join(";");
							if(vm.addType == "black"){
								url = "${ctx}/son/access/addAccessBlackList.action";
								target = vm.$refs.ctableBlack;
							}
							if(vm.addType == "white"){
								url = "${ctx}/son/access/addAccessWhiteList.action";
								target = vm.$refs.ctableWhite;
							}
							axios.post(url,stringify({serialNumber:serialNumber})).then(function(response){
								var data = response.data;
								if(data["success"]){
									target.refresh();
									vm.closeAddList();
									vm.$message({
			    						message:"<%=rb.getString("ChengGong")%>",
			    						type:'success'
			    					})
								}else{
									vm.$message.error(data["message"])
								}
							}).catch(function(){
								
							})
						}
					})
				}else{//批量导入
					vm.$refs.addListForm.validate((valid) => {
						if(valid){
							var files = document.querySelector("#uploadFileList").files;
							$("#uploadForm_list [name=fileSize]").val(files[0].size);
							if(vm.addType == "black"){
								vm.result_params = {
										operType : 'black',
										importType: vm.addListForm.import_type
								}
								var resultUrl = "${ctx}/son/access/queryImportBlackPageList.action"
					    		$("#uploadForm_list [name=operType]").val('black');
							}
							if(vm.addType == "white"){
								vm.result_params = {
										operType : 'white',
										importType: vm.addListForm.import_type
								}
								var resultUrl = "${ctx}/son/access/queryImportWhitePageList.action"
								$("#uploadForm_list [name=operType]").val('white');
								target = vm.$refs.ctableWhite;
							}
							uploadWithProgress({
								url: "${ctx}/son/access/importAccessFile.action",
								form: document.querySelector("#uploadForm_list"),
								success: function(data){
									if(data["success"]){
							     		vm.closeAddList();
							     		$("#uploadForm_list input[name='uploadFile']").val("");
							     		vm.resultVisible = true;
							     		vm.resultUrl = resultUrl;
							     		vm.count = data.count;
							     		vm.legalCount = data.legalCount;
							     		vm.unlegalCount = data.unlegalCount;
							     	}else{
						        		vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
							     	}
								}
							});
						}
					})
				}
			},
			importFile(){
				$("#uploadForm_list input[name='uploadFile']").click();
			},
			exportTempList(){
				var params = {
						tempType:'sn'
				};
				var url = "${ctx}/son/access/importAccessControlTemp.action";
				exportByForm(url,params);
			},
			addBlackList(){
				this.addType = 'black';
				this.listVisible = true;
				this.handListFlag = true;
				this.dialogTitle = 'Add eNB';
			},
			importBlackList(){
				this.addType = 'black';
				this.listVisible = true;
				this.handListFlag = false;
				this.dialogTitle = 'Import eNB';
			},
			addWhiteList(){
				this.listVisible = true;
				this.addType = 'white';
				this.handListFlag = true;
				this.dialogTitle = 'Add eNB';
			},
			importWhiteList(){
				this.addType = 'white';
				this.listVisible = true;
				this.handListFlag = false;
				this.dialogTitle = 'Import eNB';
			},
			saveResult(){
				var vm = this;
				axios.post("${ctx}/son/access/importSaveTemp.action",stringify(this.result_params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.resultVisible = false;
						if(vm.addType == "black"){
							vm.$refs.ctableBlack.refresh();
						}else{
							vm.$refs.ctableWhite.refresh();
						}
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			cancelResult(){
				this.resultVisible = false;
			},
			deleteSn(singleType,sn,hostName,listType){
				var url = "";
				var vm = this;
				var serialNumber;
				var target;
				if(listType == "black"){
					url = "${ctx}/son/access/delAccessBlackBySN.action";
					target = vm.$refs.ctableBlack;
					if(singleType == "single"){
						serialNumber = sn;
					}else{
						serialNumber = vm.$refs.ctableBlack.getChecked().toString();
					}
					var confirmStr = '<%=rb.getString("QueDingShanChuSheBei")%>';
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						closeOnClickModal:false
					}).then(() => {
						axios.post(url,stringify({serialNumber:serialNumber})).then(function(response){
							var data = response.data;
							var message = '<%=rb.getString("ChengGong")%>';
							if(data["success"]){
								vm.$message({
		    						message:message,
		    						type:'success',
		    					})
		    					target.refresh();
							}else{
								vm.$message.error(data["message"])
							}
						})
					}).catch(() => {
						
					})
				}else{
					if(singleType == "single"){
						serialNumber = sn;
						vm.hostNameList = hostName;
					}else{
						serialNumber = vm.$refs.ctableWhite.getChecked().toString();
						vm.hostNameList = vm.selectWhiteList.map(function(row){ return row.hostName;}).join(',');
					}
					vm.serialNumberList = serialNumber;
					vm.showDeleteWhiteList = true;					
				}
			},
			saveWhiteList(){
				var vm = this;
				axios.post('${ctx}/son/access/delAccessWhiteBySN.action',stringify({serialNumber:vm.serialNumberList})).then(function(response){
					var data = response.data;
					var message = '<%=rb.getString("ChengGong")%>';
					if(data["success"]){
						vm.$message({
    						message:message,
    						type:'success',
    					})
    					vm.$refs.ctableWhite.refresh();
					}else{
						vm.$message.error(data["message"])
					}
					vm.closeWhiteList();
				})
			},
			closeWhiteList(){
				var vm = this;
				vm.serialNumberList = '';
				vm.hostNameList = '';
				vm.showDeleteWhiteList = false;	
			},

			cancelDelete(){
				this.showAllSelect = false;
				this.showSelect = true;
				this.$refs.ctableBlack.clearSelection();
			},
			statusFmt(row,column,cellValue,index){
				if(cellValue == "legal"){
					return '<%=rb.getString("ChengGong")%>'
				}else{
					return '<%=rb.getString("ShiBai")%>'
				}
			},
			loadSuccessBlack(){
				this.$refs.ctableBlack.clearSelection();
			},
			loadSuccessWhite(){
				this.$refs.ctableWhite.clearSelection();
			},
			closeList(){
				accessVue.$refs.slide.hide();
			}
		},
		mounted(){
			var vm = this;
			vm.init();
			$("#uploadForm_list input[name='uploadFile']").bind("change", function() {
				vm.addListForm.file_path = this.value
			});
		},
	})
</script>
