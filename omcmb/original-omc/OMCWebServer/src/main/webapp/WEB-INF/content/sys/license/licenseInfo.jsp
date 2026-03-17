<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo ui = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<style>
#licensePage .form-item {
	flex-direction:row;
	font-size:14px;
}

#licensePage .form-item label{
	color :#5A7B92;
	margin-right:5px;
}

#devicesNum {
	padding:0px 70px;
	margin:20px 0;
	line-height:70px;
	font-size:14px;
	font-weight:600;
}

#devicesNum > div {
	width:170px;
	height:40px;
	border:1px solid var(--main-color);
	border-radius:4px;
	display:flex;
	padding:15px;
	margin-right:40px;
}
#licensePage .deviceType{
	flex:3 auto;
	border-right:1px solid var(--main-color);
}

#licensePage .deviceType span{
	width:40px;
	height:40px;
	display:inline-block;
	text-align:center;
	line-height:40px;
	border-radius:50%;
	color:#FFF;
	font-size:14px;
}

#licensePage .deviceNum {
	flex:6 auto;
}
#licensePage .deviceNum span{
	display:block;
	line-height:1;
	text-align:center;
	height:50%;
}
#licensePage .deviceNum span:first-of-type{
	color:#999;
	font-weight:normal;
}

#licensePage .listCont{
	width:80%;
	display:flex;
	height:auto;
	border:1px solid var(--main-color);
	border-radius:5px;
	margin-bottom:15px;
}
#licensePage .listCont > div:nth-of-type(1){
	display:flex;
	flex-direction:column;
	justify-content:center;
	width:80px;
	height:auto;
	text-align:center;
	border-right:1px solid var(--main-color);
	background-color: rgba(var(--main-color-rgba1),0.08);
}
#licensePage .listCont >div:nth-of-type(2){
	display:inline-block;
	width:calc(100% - 60px);
	padding:20px 20px 0 20px;
}
#licensePage .info-group {
	margin-bottom:25px;
}
#licensePage .info-title {
	border-bottom:1px solid #ddd;
	position:relative;
	margin-bottom:20px;
}
#licensePage .title-name{
	position:absolute;
	top:-10px;
	background:#FFFFFF;
	padding-right:20px;
}
#licensePage .title-name i{
	font-style:normal;
	margin-left:5px;
	color:#999;
}
#licensePage .info-text span{
	padding:0 20px;
	border:1px solid #E3EBF5;
	background-color: rgba(var(--main-color-rgba1),0.08);
	border-radius: 15px / 15px;
	width:auto;
	height:20px;
	line-height:20px;
}
#licensePage .info-text{
	display:flex;
	flex-wrap:wrap;
}
#licensePage .featureBox{
	display:flex;
	width:25%;
	margin-bottom:15px;
}
#licensePage .fontStyle{
	margin-left:10px;
	color:#1da3fc;
}
#licensePage .group-title {
	padding:20px 0 20px 40px;
}
#licensePage .form-group{
	display: block
}
</style>
<div class="overflow-cls">
<div id="licensePage" class="panelDefault" style="display:flex;flex-direction:column;min-width: 900px;overflow: hidden;">
<div class="slideBody" style="background:#FFF;overflow:auto;">
	<!--  分组  --  基本信息 -->
	<div class="form-group">
		<div class="group-title not-extend">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<div class="form-wrap">
			<div class="form-item">
				<label><%=rb.getString("LicenseID")%> : </label><span>{{id}}</span>
			</div>
			<div class="form-item">
				<label><%=rb.getString("LicenseYongTu")%> : </label><span>{{type}}</span>
			</div>
			<div class="form-item">
				<label><%=rb.getString("YouXiaoQi")%> : </label><span>{{expiryDate}}</span> 
				<span class="fontStyle">( <%=rb.getString("ShengYu")%>  <strong>{{expiryDays}}</strong>  <%=rb.getString("TianFuShu")%> )</span>
			</div>
		</div>
	</div>
	<!--  分组  --  设备支持数量 -->
	<div class="form-group">
		<div class="group-title not-extend">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiZhiChi")%></span>
		</div>
		<div id="devicesNum" class="form-wrap">

			<div v-for="item in deviceSupport">
				<div class="deviceType"><span>{{item.name}}</span></div>
				<div class="deviceNum">
					<span><%=rb.getString("ShuLiang")%></span><span>{{item.num}}</span>
				</div>
			</div>
		</div>
	</div>
	<!--  分组  --  特性列表 -->
	<div class="form-group last">
		<div class="group-title not-extend">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("NengLiLieBiao")%></span>
		</div>		
		
		<div id="featureList" class="form-wrap">
			<div class="listCont" v-for="(featureGroup,key) in newFeatureList">
				<div>
					<span>{{key}}</span>
				</div>
				<div v-if="Object.keys(newFeatureList[key]).length == '0'">
					<div class="info-group"  style="margin-bottom:0;">
						<div class="info-text">
							<div class="featureBox">
								<span><%=rb.getString("QuanBu")%></span>
							</div>
						</div>
					</div>
				</div>
				
				<div>
					<div class="info-group" v-for="(feature , group) in featureGroup">
						<div class="info-title" v-if="group != 'null'"> 
							<span class="title-name">{{group}}<i>( {{feature.length}} )</i></span>
						</div>
						<div class="info-text">
							
							<div class="featureBox" v-for="item in feature">
								<span>{{item}}</span>
							</div>
							
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
<div class="slideFooter CODE_SYSTEM_LICENSE hidden">
	<el-button style="margin-top:10px;" type="primary" @click='showImportBox = true'><%=rb.getString("GengXin")%></el-button>
</div>

<!-- 文件上传 - 更新License信息  -->
<el-dialog title='<%=rb.getString("GengXinLicense")%>' :visible.sync="showImportBox" width="400" 
     :close-on-click-modal="false" :before-close="closeFileSelect" style="margin-top:26vh;">
	<el-upload :on-change="fileChange" :on-success='checkFile' :show-file-list=false ref="upload" 
	     action="${ctx}/sys/au/uploadLicenseFile.action" :data="fileParams" name="uploadFile" :auto-upload="false">
		<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:360px;">
			<a slot="append" class="el-icon el-icon-operation-import" @click="fileSelect"></a>
		</el-input>
		<div slot="tip" class="el-upload__tip" v-show="!typeFlag"><%=rb.getString("ZhiZhiChiLICWenJian")%></div>
		<div slot="tip" class="el-upload__tip" v-show="selectFlag"><%=rb.getString("QingXianXuanZeWenJian")%></div>
		<a slot="trigger" ref="file_up"></a>
	</el-upload>
	<span slot="footer" class="dialog-footer">
		<div class="buttonGroup">
			<el-button type="primary" @click="updateLicenseInfo"><%=rb.getString("QueDing")%></el-button>
			<el-button  @click="closeFileSelect"><%=rb.getString("QuXiao")%></el-button>
		</div>	
	</span>
</el-dialog>
</div>
</div>

<script>
var license = new Vue({
	el:'#licensePage',
	data:{
		id:'',
		type:'',
		expiryDate:'',
		expiryDays:'',
		fileName:'',
		deviceSupport:[],
		featureList:[],
		newFeatureList:{},
		showImportBox:false,     //选择更新的License文件窗口，默认不显示       
		typeFlag:true,           //校验已选择的license文件格式 
		selectFlag:false,        //标识是否选择了文件  
		fileParams:{}            //上传文件时自定义的参数   
	},
	mounted(){
		this.init();
	},
	methods:{
		// 初始化获取License信息 
		init(){
			var vm = this;
			//获取License信息 
			axios.post('${ctx}/sys/au/getLicenseInfo.action',stringify({
				
			})).then(function(response){
				let data = response.data;

				var feature = data.feature_list,
					 newFeatureList={},
					 deviceSupport = [],
					 el = data;
				
				//将能力列表的数据进行二次处理 
				for ( var i=0;i<feature.length;i++){
					var name = feature[i].name;
					var child = feature[i].children;
					newFeatureList[name] = {};
					if(child.length != 0){
						newFeatureList[name][child[0].groupName]=[];
						for ( j = 1 ; j<child.length;j++){
							if (child[j].groupName == child[j-1].groupName){
								
							}else{
								newFeatureList[name][child[j].groupName]=[];
							}
						}

						for ( j = 0 ; j<child.length;j++){
							var thirdChild = child[j].children||[];
							
							if(thirdChild.length) {
								thirdChild.map(function(item){
									newFeatureList[name][child[j].groupName].push(item.name);
								});
							}else {
								newFeatureList[name][child[j].groupName].push(child[j].name);
							}
						}
					}
						
				}

				for (var item in el){
					var temp = {};
					if ( item.indexOf("support_num_") != -1){
						var deviceType = item.slice(12);
						temp.name = deviceType;
						temp.num = el[item];
						deviceSupport.push(temp)
					}
				}

				vm.id = data.license_ID;
				vm.type = data.license_type;
				vm.expiryDate = data.license_expiry_date;
				vm.expiryDays = data.license_expiry_days;
				vm.deviceSupport = deviceSupport;
				vm.newFeatureList =newFeatureList;
				
				
				vm.$refs.slide.showSlide(function(){
	    	    	vm.modal = false
	    	    });
			}).catch(function(error){})
		},
		/**
		* 选择文件后，校验格式，并赋值页面显示 
		* @param file{object}   文件信息
		* @param fileList{Array}  文件列表
		*/ 
		fileChange(file,fileList){ 
			var vm = this;
			
			vm.selectFlag = false;
			const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.lic'
			vm.typeFlag = typeFlag;
			
			if(typeFlag){
				vm.fileName = file.name;
				vm.fileParams.FileName = file.name
			}else {
				vm.fileName = '';
			}
		},
		// 选择文件
		fileSelect(){  
			var vm =this;
			vm.$refs.upload.clearFiles();
			vm.$refs['file_up'].click();
		},
		// 取消文件上传
		closeFileSelect(){
			var vm = this;
			vm.showImportBox = false;
			vm.fileName = '';
			vm.typeFlag = true;
			vm.selectFlag = false;
			vm.$refs.upload.clearFiles();
		},
		// 上传
		updateLicenseInfo(){
			var vm = this;
			if(vm.fileParams.FileName){
				vm.$refs.upload.submit();
			}else{
				vm.typeFlag = true;
				vm.selectFlag = true;
			}
			
		},
		/**
		* 文件上传成功函数 
		* @param res{object}   返回信息
		* @param file{object}  文件信息
		*/
		checkFile(res,file){    //发送请求，校验license文件内容 
			var vm = this;

			if(res.success){
				this.$confirm('<%=rb.getString("WenJianYouXiaoShiFouJiXu")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancalButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning'
				}).then(()=>{
					
					vm.showImportBox = false;
					vm.setLicense();
					vm.closeFileSelect();
				}).catch(()=>{
					
				})
			}else{
				this.$message.error(res.message)
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.upload.uploadFiles;
			fileList.forEach(function(file){
				file.status = 'ready';
			})
		},
		//下发新的license  
		setLicense(){   
			var vm = this;
			axios.post('${ctx}/sys/au/updateLicenseFile.action',stringify({
			
			})).then(function(response){
				let data = response.data;
				if(data.success){
					vm.$message({
						message:data.message,
						type:'success'
						});
						
					setTimeout(function(){
						window.location.href="${ctx}/sys/login/reloadAction.action?token=" + omctoken.replaceAll('+','%2B');
					},3000)
				}else{
					vm.$message.error(data.message)
				}
				

			}).catch(function(error){})
			 
			vm.fileName = '';
			vm.$refs.upload.clearFiles();
		}
	}
	
});

</script>

