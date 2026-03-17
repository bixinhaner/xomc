<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.licEdit{
		margin-bottom:30px;
	}
	.licEdit .el-input{
		width:350px;
		height:26px;
	}
	.licEdit label{
		display:block;
		margin-bottom:8px;
		font-size:14px;
		color:#5A7B92;
	}
	.slide-content{
		margin-left:50px;
	}
	.licEdit .el-textarea{
		width:350px;
	}
	.licEdit .el-textarea__inner{
		font-size:12px;
	}
</style>
<div id='modifyLicense' style='padding-top:20px;'>
	<div class='licEdit'>
		<label>1588 License File</label>
		<el-input v-model='file_name' disabled=true></el-input>
	</div>
	<div class='licEdit'>
		<label><%=rb.getString("MACDiZhi")%></label>
		<el-input type='textarea' autosize v-model='mac_range' disabled=true></el-input>
	</div>
	<div class='licEdit'>
		<label><%=rb.getString("WenJianDaXiao")%></label>
		<el-input v-model='file_size' disabled=true></el-input>
	</div>
	<div class='licEdit'>
		<label><%=rb.getString("ShangChuanRen")%></label>
		<el-input v-model='uploader' disabled=true></el-input>
	</div>
	<div class='licEdit'>
		<label><%=rb.getString("ShangChuanShiJian")%></label>
		<el-input v-model='upload_time' disabled=true></el-input>
	</div>
	<div class='licEdit'>
		<label><%=rb.getString("MiaoShu")%></label>
		<el-input v-model='description' type='textarea' :autosize="{minRows:4}"></el-input>
	</div>
</div>
<script>
	new Vue({
		el:'#modifyLicense',
		data(){
			return{
				file_name:'',
				mac_range:'',
				file_size:'',
				description:'',
				uploader:'',
				upload_time:'',
				defaultDesc:'',
				file_id:''
			}
		},
		methods:{
			getFileInfo(file_id){
				var vm = this;
				vm.file_id = file_id;
				axios.post('${ctx}/cell/1588License/getFileInfoById.action',stringify({
					file_id : file_id,
					time_zone : timeZone
				})).then(function(response){
					var data = response.data.rows[0];
					vm.file_name = data.file_name;
					vm.mac_range = data.mac_section.replace(/,/g,',\n');
					vm.file_size = data.file_size;
					vm.description = data.description;
					vm.defaultDesc = data.description;
					vm.uploader = data.uploader;
					vm.upload_time = data.upload_time;
				})
			},
			submit(){
				var vm = this;
				axios.post('${ctx}/cell/1588License/modifyFileInfo.action',stringify({
					file_id : vm.file_id,
					description:vm.description
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message({
    						message:'<%=rb.getString("ChengGong")%>',
    						type:'success',
    					})
                        eventBus.$emit('save-file-suc')
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			cancelFile(){
				var vm = this;
				if(vm.description != vm.defaultDesc){
					var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						eventBus.$emit('hide-file')
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-file')
				}
			}
		},
		mounted(){
			eventBus.$off('edit-file').$on('edit-file',this.getFileInfo)
			eventBus.$off('save-lic').$on('save-lic',this.submit)
			eventBus.$off('cancel-file').$on('cancel-file',this.cancelFile)
		}
	})
</script>
