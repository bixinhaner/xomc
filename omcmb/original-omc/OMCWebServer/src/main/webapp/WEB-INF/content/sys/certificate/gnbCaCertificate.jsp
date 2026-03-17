<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
	.cacertWarp{
		height: 100%;
		display: flex;
		width:100%;
		position:relative;
		overflow:hidden;
		background: #F6F7FB;
	}
	.cacertWarp .w320{
		width:320px;
	}
</style>

<div id="gnbCaCertificatePage" class="cacertWarp">
	<div class='leftWarp commonWarp'>
		<!--操作按钮  -->
		<div v-show="optBtnShow" class="circleIcon placeholder-bt" style="right: 50px; top: 42px" placeholder="<%=rb.getString("ZhengShuDaoRu")%>">		
			<span class="el-icon el-icon-circle-import" @click="importCaCertificate"></span>
		</div>
		<div class="circleIcon placeholder-bt" style="right: 10px; top: 42px" placeholder="">		
			<span class="el-icon el-icon-circle-close" @click="closeCaSlider"></span>
		</div>
		<div class='leftBoxHeader'><%=rb.getString("CAZhengShu")%></div>
		
		<el-ctable ref="caCertsTable" :url="caCertsUrl" :query-params="caCretsParams" id="caCertTable" :page-size="pageSize" pagination="true" 
			:limit="limitBatch" :rownumber=true :row-key="'id'" style="width:100%" @selection-change='caBatchSelect'>
			<template slot="toolbar">
			
				<div class='toolbarHeadBtnBoxCls' style='margin: -10px 0 0 0;'>
	                <!-- 已选数据 -->
	            	<div class="selectBlukBoxCls">
	                	<div class="selectMain headBtnItemCls">
	                        <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable" style='border-right: 0; padding: 0;'>
								<span class="el-icon-selected el-icon"></span>
								<span class="bulkSelectNumBoxCls">( {{caCertSelectData.length}} )</span>
							</div>
							<div class="selectTableBoxCls" v-show="bulkSelectShow" style="position: absolute;top: 32px;left: 30px;">
								<div class="selectBoxTitle">
									<span><%=rb.getString("YiXuan")%></span>
	                                <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
	                            </div>
	                            <div class="selectBoxMain">
									<div class="tableInfoCls">
	                                    <div class="tableInfoHeader">
											<div><%=rb.getString("YiXuanWenJian")%></div>
	                                        <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
	                                    </div>
	                                    <el-ctable 
											id="bulkSelectTable" 
											ref="bulkSelectTable" 
											:data="caCertSelectData" 
											:showHeader="false"
											:rownumber="false"
											:front-pagination="true"
											height="270px" pagination="true" >
											<el-table-column prop="id" v-if="false"></el-table-column>
											<el-table-column width="588">
												<template slot-scope="scope" >
													<div class="tableItemCls">
														<span>{{scope.row.caCertName}}</span>
														<span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
													</div>
												</template>
											</el-table-column>
										</el-ctable>
	                                </div>
	                            </div>
	                        </div>
	                    </div>
	                </div>
	
					<div :class="caCertSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="caCertsDownloadBatch">
						<span class='el-icon el-icon-operation-download'></span>
						<span><%=rb.getString("PiLiangXiaZai")%></span>
					</div>
					<div v-show="optBtnShow" :class="caCertSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="caCertsDeleteBatch" style='border-right: 0;'>
						<span class='el-icon el-icon-operation-delete'></span>
						<span><%=rb.getString("PiLiangShanChu")%></span>
					</div>
	            </div>
				<div class='commonQuery' style=' border: 0; margin-top: 10px;'>
					<el-query type="normal" @query="caCaertsQuery" placeholder="<%=rb.getString("ZhengShuWenJian")%>"></el-query>                      
				</div>
	        </template>
	        <el-table-column type="selection" :reserve-selection="true" ></el-table-column>
                  
			<el-table-column label="<%=rb.getString("ZhengShuWenJian")%>" prop="fileName"></el-table-column>                        
			<el-table-column label="<%=rb.getString("ShangChuanRen")%>" prop="uploader"></el-table-column> 
			<el-table-column label="<%=rb.getString("ShangChuanShiJian")%>" prop="uploadTime"></el-table-column>
			<el-table-column label="<%=rb.getString("WangGuanMiaoShu")%>" prop="description"></el-table-column>                           
	     </el-ctable>
	</div>
	<!-- 导入文件框 -->
	<div class='rightWarp' style='position: relative; flex: 0 1 360px;' v-show='caCertsshowImportCard'>
		<div class='rightWarpLayer'>
	 		<div class='rightBoxHeaderHasTip'>
				<div class='headerText'>
					<span><%=rb.getString("ZhengShuDaoRu")%></span>
					<span class='closeIconBox' @click='caCertsCloseImport'><i class='el-icon el-icon-close'></i></span>
				</div>
			</div>
			<div class='rightWarpLayerContent'>
				<el-form label-position="top" ref="caRuleForm" :model='caRuleForm' :rules='rules' style='padding: 30px 20px;'>     					      
	                <el-form-item label="<%=rb.getString("CAZhengShu")%>" prop="fileName">
	                  	<el-upload :before-upload='beforeUpload' :on-success='checkFile' :on-change="fileChange"  :show-file-list=false ref="upload"
						     :action="caRuleForm.uploadFileUrl" :data="fileParams" name="uploadFile" :auto-upload="false">
							<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w320">
								<a slot="append" class="el-icon el-icon-operation-import" @click="fileSelect"></a>
							</el-input>								
							<a slot="trigger" ref="file_up"></a>
						</el-upload>
                    </el-form-item>      
                    <el-form-item label="<%=rb.getString("WangGuanMiaoShu")%>" prop='description'>
                        <el-input v-model='caRuleForm.description' type='textarea' :rows='4' class="w270"></el-input>
                    </el-form-item> 		                
	            </el-form> 
			</div>
			<div class='commonFlex commonBorderTop commonFormFotter'>
				<div>
					<el-button type="primary" @click="caCertsUploadImport"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="caCertsCloseImport"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
		</div>
	</div>	
</div>

<script type="text/javascript">
	var gnbCaCertificateVue = new Vue({
	    el: '#gnbCaCertificatePage',
	    data() {
	    	var vm = this,
	    		validateCaName = function(rule,value,callback) {
	            	value = vm.fileName;     		
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}else {
						callback();
					}
				};
	    	return {
	    		pageSize:50,
				caCertsUrl: "${ctx}/cell/cert/getIpsecCACertInfoPageList.action",
	    		caCretsParams: {
	    			timeZone: timeZone,
	             	searchText: '',
	             	nbType: 'gNB'
	            }, 
	           	caCertSelectData: [],
	           	caCertsshowImportCard: false,
	            tableSelect: false,	           
	            caRuleForm: {
	                uploadFileUrl: '',
	               	description:'',
	           	},	         	          
	            fileParams: {},              
	            fileName: '',	            					
				fileList: [],
				rules: {	           		
					fileName: [
                    	{required: true, validator: validateCaName},
                    ]                   
	            },
	            bulkSelectShow: false
	    	}
	    },
		computed: {
			limitBatch(){
				return batchOperation ? '' : 1;
			},
			optBtnShow() {
				return writableMap['CODE_GNB_IPSEC_CERT'] == true;
			},
	    },
		methods: {
	    	/**
			* 文件上传成功函数 
			* @param res{object}   返回信息
			* @param file{object}  文件信息
			* 发送请求，校验device文件内容 
			*/
			checkFile(res, file){    
				var vm = this;
				
				if(res.success){
					if(res.suc_count > 0){
						vm.$message({
							type: 'success',
							message: '<%=rb.getString("ChengGong")%>'
						});
					}else {
						vm.$message({
							type: 'warning',
							message: '<%=rb.getString("ShiBai")%>'
						});
					}
					vm.caCertsshowImportCard = false;
					vm.$refs.caCertsTable.refresh();
					vm.closeFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态  
				var fileList = vm.$refs.upload.uploadFiles;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
	    	
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			fileChange(file, fileList){ 
				var vm = this;
				
				vm.fileName = file.name;
				vm.fileParams.FileName = file.name;
			},
			
			// 选择文件
			fileSelect(){  
				var vm =this;
				
				vm.$refs.upload.clearFiles();
				vm.$refs['file_up'].click();
			},
			
			// 移除导入文件
			closeFileSelect(){
				var vm = this;
				
				vm.fileName = '';			
				vm.$refs.upload.clearFiles();
			},
						
			/**
			* 文件上传之前
			* @param file{object}   文件信息
			*/ 
			beforeUpload(file){
				var vm = this,
					fileName = file.name,
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};
				fd.append('uploadFile', file); //文件流
				fd.append('fileName', fileName);//文件名
				fd.append('description', vm.caRuleForm.description);//描述
				fd.append('nbType', 'gNB');//文件名
				axios.post("${ctx}/cell/cert/uploadIpsecCACertFile.action",fd,config).then(function(res){
					if(res.data["success"]){	
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.$refs.caCertsTable.refresh();
						
						vm.caCertsshowImportCard = false;
						vm.description = '';
						vm.fileList = [];
						vm.fileName = '';
						vm.$refs.caRuleForm.resetFields();						
					}else{
						vm.$message.error(res.data["message"])
					}
				})
				
				return false;
			},
			
			/*确定导入*/
	        caCertsUploadImport() {
				var vm = this;
				
				vm.$refs.caRuleForm.validate((valid) => {
                    if (valid) {
                    	vm.$refs.upload.submit();                   	
                    }
                }) 				
			},
		
			//导入设备证书
			importCaCertificate(){
				var vm= this;
				
				vm.caCertsshowImportCard = true;
			},
			
			// 关闭导入弹出框
			caCertsCloseImport(){
				var vm = this;
				
				vm.description = '';
				vm.fileList = [];
				vm.fileName = '';
				vm.$refs.caRuleForm.resetFields();
				vm.caCertsshowImportCard = false;
			},
											
	    	//Ipsec 模块  搜索事件 
	    	caCaertsQuery(val){
				var vm = this;
				
	    		Object.assign(vm.caCretsParams,{    			
	    			searchText: val	    		
	    		}) 
	    	},	    	

	    	/**
			* 列表选中
			* @param selection{Array}   选中数据
			*/
			caBatchSelect(selection){
				var vm = this;
				vm.caCertSelectData = selection.map((item)=>{
					return Object.assign(item,{caCertName: item.fileName});					
				});
			},
	    		    
		    //批量下载
		    caCertsDownloadBatch(){
		    	var vm = this, certIds=[];
				if(vm.caCertSelectData.length == 0){
					return
				}else{
					certIds = vm.caCertSelectData.map(function(item){ return item.id});
				}
				
				exportByForm("${ctx}/cell/cert/downLoadIpsecCACertFile.action",{		        	
		        	id: certIds.join(','),
		        	nbType: 'gNB'
		        });				
		    },
		     
		 	 //批量删除
			caCertsDeleteBatch(){
				var vm = this, certIds=[];
				
				if(vm.caCertSelectData.length == 0){
					return
				}else{
					certIds = vm.caCertSelectData.map(function(item){ return item.id});
				}
				vm.$confirm('<%=rb.getString("QueDingPiLiangShanChuCAZhengShu")%>','<%=rb.getString("ZhengShuShanChu")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
				}).then(function(){
					axios.post('${ctx}/cell/cert/deleteIpsecCACertInfo.action',stringify({
						id: certIds.join(','),
						nbType: 'gNB'
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.$refs.caCertsTable.refresh();				
						}else{							
							vm.$message.error(data["message"])
						}
						vm.caCertSelectData = [];
						vm.$refs.caCertsTable.clearSelection();
					})
				}).catch(function(){
					
				})	
				
			},
   	
	       
	        //----------------------------------------------------------- 已选 -----------------------------------------------------------
	     	// 打开已选弹窗
            openBulkSelectTable(){
                var vm = this;
                vm.bulkSelectShow = true
            },
            // 关闭已选弹窗
            closeBulkSelectTable(){
                var vm = this;
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 清空事件
            clearBulkSelected(){
                var vm = this;
				
                vm.$refs["caCertsTable"].clearSelection();
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 单个删除事件
            delBulkSelected(rows){
                var vm = this,
					tabs = 'caCertsTable',
					rowKey = 'id';

				vm.caCertSelectData = vm.caCertSelectData.filter((items)=>{
					return items[rowKey] != rows[rowKey]
				});
				var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
					irow= selection.filter((items)=>{
						return items[rowKey] == rows[rowKey]
					})[0];
				vm.$refs[tabs].toggleRowSelection(irow,false);
				var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
				vm.$refs[tabs].ckList.splice(idx,1);
                
                if(vm.caCertSelectData.length == 0){
                	vm.bulkSelectShow = false;
                }
            },
            // 关闭 CA 证书页面
            closeCaSlider(){
            	gnbCertificateManageVue.$refs.slideCA.hide();
	        },
	    }
	});
	
</script>