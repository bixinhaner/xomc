<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
	.sslCertWarp{
		height: 100%;
		display: flex;
		width:100%;
		position:relative;
		overflow:hidden;
	}
	.sslCertWarp .w320{
		width:320px;
	}
</style>

<div id="gnbSslCertificateManage" class="sslCertWarp">
	<div class='leftWarp commonWarp'>
		<!--操作按钮  -->
		<div v-show="optBtnShow" class="circleIcon placeholder-bt" style="right:50px;top:42px" placeholder="<%=rb.getString("ZhengShuDaoRu")%>">		
			<span class="el-icon el-icon-circle-import" @click="gnbImportSSLCertificate"></span>
		</div>
		<div class="circleIcon placeholder-bt" style="right:10px;top:42px" placeholder="">		
			<span class="el-icon el-icon-circle-close" @click="gnbcloseSSLSlider"></span>
		</div>
		<div class='leftBoxHeader'><%=rb.getString("SSLZhengShu")%></div>
		
		<el-ctable ref="gnbSslCertsTable" :url="gnbSslCertsUrl" :query-params="sslCretsParams" id="sslCertTable" :page-size="pageSize" pagination="true" 
			:rownumber=true :row-key="'id'" style="width:100%" @selection-change='gnbSslBatchSelect' :limit="limitBatch">
			<template slot="toolbar">
				<div class='toolbarHeadBtnBoxCls' style='margin: -10px 0 0 0;'>
	                <!-- 已选数据 -->
	            	<div class="selectBlukBoxCls">
	                	<div class="selectMain headBtnItemCls">
	                        <div class="bulkSelectBtnBoxCls"  @click="gnbOpenBulkSelectTable" style='border-right: 0; padding: 0;'>
								<span class="el-icon-selected el-icon"></span>
								<span class="bulkSelectNumBoxCls">( {{sslCertSelectData.length}} )</span>
							</div>
							<div class="selectTableBoxCls" v-show="bulkSelectShow" style="position: absolute;top: 32px;left: 30px;">
								<div class="selectBoxTitle">
									<span><%=rb.getString("YiXuan")%></span>
	                                <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="gnbCloseBulkSelectTable"></span>
	                            </div>
	                            <div class="selectBoxMain">
									<div class="tableInfoCls">
	                                    <div class="tableInfoHeader">
											<div><%=rb.getString("YiXuanWenJian")%></div>
	                                        <div @click="gnbSslClearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
	                                    </div>
	                                    <el-ctable 
											id="bulkSelectTable" 
											ref="bulkSelectTable" 
											:data="sslCertSelectData" 
											:showHeader="false"
											:rownumber="false"
											:front-pagination="true"
											height="270px" pagination="true" >
											<el-table-column prop="id" v-if="false"></el-table-column>
											<el-table-column width="588">
												<template slot-scope="scope" >
													<div class="tableItemCls">
														<span>{{scope.row.sslCertName}}</span>
														<span @click="gnbDelBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
													</div>
												</template>
											</el-table-column>
										</el-ctable>
	                                </div>
	                            </div>
	                        </div>
	                    </div>
	                </div>
	
					<div :class="sslCertSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="gnbSslCertsDownloadBatch">
						<span class='el-icon el-icon-operation-download'></span>
						<span><%=rb.getString("PiLiangXiaZai")%></span>
					</div>
					<div v-show="optBtnShow" :class="sslCertSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="gnbSslCertsDeleteBatch" style='border-right: 0;'>
						<span class='el-icon el-icon-operation-delete'></span>
						<span><%=rb.getString("PiLiangShanChu")%></span>
					</div>
				</div>
				<div class='commonQuery' style=' border: 0; margin-top: 10px;'>
					<el-query type="normal" @query="gnbSslCertsQuery" placeholder="<%=rb.getString("ZhengShuWenJian")%>"></el-query>                      
				</div>
			</template>
			<el-table-column type="selection" :reserve-selection="true" ></el-table-column>
	        <el-table-column label="<%=rb.getString("ZhengShuWenJian")%>" prop="fileName"></el-table-column>                        
	        <el-table-column label="<%=rb.getString("ShangChuanRen")%>" prop="uploader"></el-table-column> 
	        <el-table-column label="<%=rb.getString("ShangChuanShiJian")%>" prop="uploadTime"></el-table-column>
	        <el-table-column label="<%=rb.getString("MiaoShu")%>" prop="description"></el-table-column>                           
	     </el-ctable>
	</div>
	<!-- 导入文件框 -->
	<div class='rightWarp' style='position: relative; flex: 0 1 360px;' v-show='sslCertsshowImportCard'>
		<div class='rightWarpLayer'>
	 		<div class='rightBoxHeaderHasTip'>
				<div class='headerText'>
					<span><%=rb.getString("DaoRuZhengShu")%></span>
					<span class='closeIconBox' @click='gnbSslCertsCloseImport'><i class='el-icon el-icon-close'></i></span>
				</div>
			</div>
			<div class='rightWarpLayerContent'>
				<el-form label-position="top" ref="sslRuleForm" :model='sslRuleForm' :rules='rules' style='padding: 30px 20px;'>     					      
	                <el-form-item label="<%=rb.getString("SSLZhengShu")%>" prop="fileName">
	                  	<el-upload :before-upload='gnbSslBeforeUpload' :on-success='gnbSslCheckFile' :on-change="gnbSslFileChange" :show-file-list=false ref="upload"
						     :action="sslRuleForm.uploadFileUrl" :data="fileParams" name="uploadFile" :auto-upload="false">
							<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w320">
								<a slot="append" class="el-icon el-icon-operation-import" @click="gnbSslFileSelect"></a>
							</el-input>								
							<a slot="trigger" ref="file_up"></a>
						</el-upload>
                    </el-form-item>      
                    <el-form-item label="<%=rb.getString("MiaoShu")%>" prop='description'>
                        <el-input v-model='sslRuleForm.description' type='textarea' :rows='4' class="w270"></el-input>
                    </el-form-item> 		                
	            </el-form> 
			</div>
			<div class='commonFlex commonBorderTop commonFormFotter'>
				<div>
					<el-button type="primary" @click="gnbSslCertsUploadImport"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="gnbSslCertsCloseImport"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
		</div>
	</div>	
</div>

<script type="text/javascript">
	var gnbSslCertificateManageVue = new Vue({
	    el: '#gnbSslCertificateManage',
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
	            gnbSslCertsUrl: "${ctx}/cert/ssl/getGNBSSLCertInfoPageList.action",	
	            sslCretsParams: {
	    			timeZone: timeZone,
	             	searchText: '',
	            }, 
	           	sslCertSelectData: [],
	           	sslCertsshowImportCard: false,
	            tableSelect: false,	           
	            sslRuleForm: {
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
			gnbSslCheckFile(res, file){    
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
					vm.sslCertsshowImportCard = false;
					vm.$refs.gnbSslCertsTable.refresh();
					vm.gnbSslCloseFileSelect();
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
			gnbSslFileChange(file, fileList){ 
				var vm = this;
				
				vm.fileName = file.name;
				vm.fileParams.FileName = file.name;
			},
			
			// 选择文件
			gnbSslFileSelect(){  
				var vm =this;
				
				vm.$refs.upload.clearFiles();
				vm.$refs['file_up'].click();
			},
			
			// 移除导入文件
			gnbSslCloseFileSelect(){
				var vm = this;
				
				vm.fileName = '';			
				vm.$refs.upload.clearFiles();
			},
						
			/**
			* 文件上传之前
			* @param file{object}   文件信息
			*/ 
			gnbSslBeforeUpload(file){
				var vm = this,
					fileName = file.name,
					fileSize = file.size,
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};
				fd.append('deviceType', 'gNB');
				fd.append('uploadFile', file); //文件流
				fd.append('fileName', fileName);//文件名
				fd.append('fileSize', fileSize);//大小
				fd.append('description', vm.sslRuleForm.description);//描述
				
				axios.post("${ctx}/cert/ssl/uploadSSLCertFile.action",fd,config).then(function(res){
					if(res.data["success"]){	
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.$refs.gnbSslCertsTable.refresh();
						
						vm.sslCertsshowImportCard = false;
						vm.description = '';
						vm.fileList = [];
						vm.fileName = '';
						vm.$refs.sslRuleForm.resetFields();						
					}else{
						vm.$message.error(res.data["message"])
					}
				})
				
				return false;
			},
			
			/*确定导入*/
	        gnbSslCertsUploadImport() {
				var vm = this;
				
				vm.$refs.sslRuleForm.validate((valid) => {
                    if (valid) {
                    	vm.$refs.upload.submit();                   	
                    }
                }) 				
			},
		
			//导入设备证书
			gnbImportSSLCertificate(){
				var vm= this;
				
				vm.sslCertsshowImportCard = true;
			},
			
			// 关闭导入弹出框
			gnbSslCertsCloseImport(){
				var vm = this;
				
				vm.description = '';
				vm.fileList = [];
				vm.fileName = '';
				vm.$refs.sslRuleForm.resetFields();
				vm.sslCertsshowImportCard = false;
			},
											
	    	//Ipsec 模块  搜索事件 
	    	gnbSslCertsQuery(val){
				var vm = this;
				
	    		Object.assign(vm.sslCretsParams,{    			
	    			searchText: val	    		
	    		}) 
	    	},	    	

			gnbSslBatchSelect(selection){
				var vm = this;
				
				vm.sslCertSelectData = selection.map((item)=>{
					return Object.assign(item, {sslCertName: item.fileName});					
				});
			},
	    		    
		    //批量下载
		    gnbSslCertsDownloadBatch(){
		    	var vm = this, certIds = [];
		    	
				if(vm.sslCertSelectData.length == 0){
					return
				}else{
					certIds = vm.sslCertSelectData.map(function(item){ return item.id});
				}
				
				exportByForm("${ctx}/cert/ssl/downLoadSSLCertFile.action",{		        	
		        	id: certIds.join(','),
		        	timeZone: timeZone
		        });				
		    },
		    
		 	//批量删除
			gnbSslCertsDeleteBatch(){
				var vm = this, certIds=[];
				
				if(vm.sslCertSelectData.length == 0){
					return
				}else{
					certIds = vm.sslCertSelectData.map(function(item){ return item.id});
				}
				
				vm.$confirm('<%=rb.getString("QueDingPiLiangShanChuSSLZhengShu")%>','<%=rb.getString("ZhengShuShanChu")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
				}).then(function(){
					axios.post('${ctx}/cert/ssl/deleteSSLCertInfo.action',stringify({
						id : certIds.join(','),
					})).then(function(response){
						var data = response.data;
						
						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.$refs.gnbSslCertsTable.refresh();						
						}else{							
							vm.$message.error(data["message"]);
						}
						
						vm.sslCertSelectData = [];
						vm.$refs.gnbSslCertsTable.clearSelection();
					})
				}).catch(function(){})	
			},
   			//关闭SSL证书维护页面
	        gnbcloseSSLSlider(){
	        	gnbCertificateManageVue.$refs.gnbSlideCertManagePage.hide();
	        },
	        //--------------------------------------------------------------已选
	     	// 打开已选弹窗
            gnbOpenBulkSelectTable(){
                var vm = this;
                
                vm.bulkSelectShow = true;
            },
            // 关闭已选弹窗
            gnbCloseBulkSelectTable(){
                var vm = this;
                
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 清空事件
            gnbSslClearBulkSelected(){
                var vm = this;
				
                vm.$refs["gnbSslCertsTable"].clearSelection();
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 单个删除事件
            gnbDelBulkSelected(rows){
                var vm = this,
					tabs = 'gnbSslCertsTable',
					rowKey = 'id';
				vm.sslCertSelectData = vm.sslCertSelectData.filter((items)=>{
					return items[rowKey] != rows[rowKey]
				});
				var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
					irow= selection.filter((items)=>{
						return items[rowKey] == rows[rowKey]
					})[0];
				vm.$refs[tabs].toggleRowSelection(irow,false);
				var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
				vm.$refs[tabs].ckList.splice(idx,1);
                
                if(vm.sslCertSelectData.length == 0){
                	vm.bulkSelectShow = false;
                }
            }
	    },
	});
	
</script>
